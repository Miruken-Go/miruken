package httpsrv

import (
	context2 "context"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Handler augments http.Handler to provide miruken.Handler composer.
	Handler interface {
		ServeHTTP(
			w http.ResponseWriter,
			r *http.Request,
			h miruken.Handler,
		)
	}

	// HandlerFunc promotes a function to Handler.
	HandlerFunc func(
		w http.ResponseWriter,
		r *http.Request,
		h miruken.Handler,
	)
)

func (f HandlerFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
) {
	f(w, r, h)
}

// Use builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating
// at a provided handler.
func Use(
	ctx        *context.Context,
	handler    any,
	middleware ...any,
) http.Handler {
	pipeline := Pipe(middleware...)

	switch h := handler.(type) {
	case http.Handler:
		return &rawAHandler{baseHandler{ctx, pipeline}, h}
	case Handler:
		return &extHandler{baseHandler{ctx, pipeline}, h}
	case func(http.ResponseWriter, *http.Request):
		return &rawAHandler{baseHandler{ctx, pipeline}, http.HandlerFunc(h)}
	case func(http.ResponseWriter, *http.Request, miruken.Handler):
		return &extHandler{baseHandler{ctx, pipeline}, HandlerFunc(h)}
	default:
		fun := &funHandler{baseHandler: baseHandler{ctx, pipeline}}
		if fun.tryBind(handler) {
			return fun
		}
		panic(fmt.Errorf(
			"httpsrv: %T is not a http.Context, httpsrv.Context or compatible handler function",
			handler))
	}
}

// Api builds a http.Handler for processing polymorphic api calls
// through a Middleware pipeline.
func Api(
	ctx        *context.Context,
	middleware ...any,
) http.Handler {
	return Use(ctx, H[*PolyHandler](), middleware...)
}

// H builds a typed Handler wrapper for dynamic http processing.
func H[H any](opts ...any) Handler {
	typ := reflect.TypeFor[H]()
	if typ.Implements(handlerType) {
		return &rawResHandler{resHandler[http.Handler]{typ: typ, opts: opts}}
	}
	if typ.Implements(extHandlerType) {
		return &extResHandler{resHandler[Handler]{typ: typ, opts: opts}}
	}
	if binding, err := getHandlerBinding(typ); err == nil {
		return &dynResHandler{resHandler[any]{typ: typ, opts: opts}, binding}
	}
	panic(fmt.Errorf(
		"httpsrv: %v is not a http.Context, httpsrv.Context or compatible handler type", typ))
}

// getHandlerBinding discovers a suitable handler ServerHTTP method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getHandlerBinding(
	typ reflect.Type,
) (handlerBinding, error) {
	if bindings := handlerBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return binding, nil
		}
	}
	handlerLock.Lock()
	defer handlerLock.Unlock()
	bindings := handlerBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return binding, nil
		}
		sb := maps.Clone(*bindings)
		bindings = &sb
	} else {
		bindings = &map[reflect.Type]handlerBinding{}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		binding, err := makeHandlerBinding(method.Type, method.Func, 1)
		if binding != nil {
			(*bindings)[typ] = binding
			handlerBindingMap.Store(bindings)
			return binding, nil
		} else if err != nil {
			return nil, &miruken.MethodBindingError{Method: &method, Cause: err}
		}
	}
	return nil, fmt.Errorf(`httpsrv: handler %v has no compatible dynamic method`, typ)
}

func makeHandlerBinding(
	typ reflect.Type,
	fun reflect.Value,
	idx int,
) (handlerBinding, error) {
	if typ.NumIn() < 2 || typ.NumOut() > 0 {
		return nil, nil
	}
	for i := 0; i < 2; i++ {
		if typ.In(i+idx) != handlerFuncType.In(i) {
			continue
		}
	}
	if caller, err := miruken.MakeCaller(fun); err != nil {
		return nil, err
	} else {
		return handlerBinding(caller), nil
	}
}

type (
	// baseHandler provides common behavior for adapting a pipeline.
	// to various http handler mechanisms.
	baseHandler struct {
		ctx      *context.Context
		pipeline Middleware
	}

	// rawAHandler adapts a standard http.Handler to a pipeline.
	rawAHandler struct {
		baseHandler
		handler http.Handler
	}

	// extHandler adapts an enhanced Handler to a pipeline.
	extHandler struct {
		baseHandler
		handler Handler
	}

	// funHandler adapts a handler function to a pipeline.
	funHandler struct {
		baseHandler
		binding handlerBinding
	}

	// resHandler adapts a resolved handler to a pipeline.
	resHandler[H any] struct {
		typ  reflect.Type
		opts []any
	}

	// rawResHandler adapts a resolved http.Handler to a pipeline.
	rawResHandler struct {
		resHandler[http.Handler]
	}

	// extResHandler adapts a resolved Handler to a pipeline.
	extResHandler struct {
		resHandler[Handler]
	}

	// dynResHandler adapts a resolved compatible handler to a pipeline.
	dynResHandler struct {
		resHandler[any]
		binding handlerBinding
	}

	// handlerBinding represents the method used by a
	// dynamic handler to serve http content.
	handlerBinding miruken.CallerFunc

	// Specialized key type for context values.
	contextKey int
)

// ComposerKey is used to access the miruken.Handler from the context.
const ComposerKey contextKey = 0

func (a *baseHandler) serve(
	w http.ResponseWriter,
	r *http.Request,
	h func(c miruken.Handler),
) {
	defer handlePanic(w, r)
	child := a.ctx.NewChild()
	defer child.Dispose()

	a.pipeline.ServeHTTP(w, r, child, func(c miruken.Handler) {
		if c == nil {
			c = child
		}
		h(c)
	})
}

func (a *rawAHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		ctxWithComposer := context2.WithValue(r.Context(), ComposerKey, c)
		a.handler.ServeHTTP(w, r.WithContext(ctxWithComposer))
	})
}

func (a *extHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		a.handler.ServeHTTP(w, r, c)
	})
}

func (a *funHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		if err := a.binding.invoke(c, w, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

func (a *funHandler) tryBind(fun any) bool {
	typ := reflect.TypeOf(fun)
	if typ.Kind() != reflect.Func {
		return false
	}
	binding, err := makeHandlerBinding(typ, reflect.ValueOf(fun), 0)
	if binding != nil && err == nil {
		a.binding = binding
		return true
	}
	return false
}

func (a *resHandler[H]) serve(
	w http.ResponseWriter,
	c miruken.Handler,
	h func(handler H),
) {
	if opts := a.opts; len(opts) > 0 {
		c = miruken.BuildUp(c, provides.With(opts...))
	}
	handler, ph, ok, err := miruken.ResolveKey[H](c, a.typ)
	if !(ok && err == nil) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if ph != nil {
		if handler, err = ph.Await(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	h(handler)
}

func (a *rawResHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	c miruken.Handler,
) {
	a.serve(w, c, func(h http.Handler) {
		ctxWithComposer := context2.WithValue(r.Context(), ComposerKey, c)
		h.ServeHTTP(w, r.WithContext(ctxWithComposer))
	})
}

func (a *extResHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	c miruken.Handler,
) {
	a.serve(w, c, func(h Handler) {
		h.ServeHTTP(w, r, c)
	})
}

func (a *dynResHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	c miruken.Handler,
) {
	a.serve(w, c, func(h any) {
		if err := a.binding.invoke(c, h, w, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

func (b handlerBinding) invoke(
	c        miruken.Handler,
	initArgs ...any,
) error {
	_, pr, err := b(c, initArgs...)
	if err != nil && pr != nil {
		_, err = pr.Await()
	}
	return err
}

var (
	handlerType    = reflect.TypeFor[http.Handler]()
	extHandlerType = reflect.TypeFor[Handler]()

	handlerLock       sync.Mutex
	handlerFuncType   = reflect.TypeFor[http.HandlerFunc]()
	handlerBindingMap = atomic.Pointer[map[reflect.Type]handlerBinding]{}
)
