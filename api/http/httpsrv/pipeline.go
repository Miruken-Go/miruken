package httpsrv

import (
	context2 "context"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/provides"
	"log"
	"maps"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
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

	// Middleware augments Handler to participate in a pipeline to support
	// pre and post processing of requests.
	Middleware interface {
		ServeHTTP(
			Middleware,
			http.ResponseWriter,
			*http.Request,
			Middleware,
			miruken.Handler,
			func(miruken.Handler),
		)
	}

	// MiddlewareFunc promotes a function to Middleware.
	MiddlewareFunc func(
		s Middleware,
		w http.ResponseWriter,
		r *http.Request,
		m Middleware,
		h miruken.Handler,
		n func(miruken.Handler),
	)

	// MiddlewareAdapter is an adapter for implementing
	// Middleware using late binding method resolution.
	MiddlewareAdapter struct {}
)


func (f HandlerFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
) { f(w, r, h) }


func (f MiddlewareFunc) ServeHTTP(
	s Middleware,
	w http.ResponseWriter,
	r *http.Request,
	m Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) { f(s, w, r, m, h, n) }


// MiddlewareAdapter

func (m MiddlewareAdapter) ServeHTTP(
	s Middleware,
	w http.ResponseWriter,
	r *http.Request,
	p Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	if binding, err := getMiddlewareMethod(s); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		initArgs := []any{s,w,r}
		for _, idx := range binding.argIndexes {
			switch idx {
			case 3:
				initArgs = append(initArgs, p)
			case 4:
				initArgs = append(initArgs, h)
			case 5:
				initArgs = append(initArgs, n)
			}
		}
		_, pr, err := binding.caller(h, initArgs...)
		if err != nil && pr != nil {
			_, err = pr.Await()
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}


// middlewareBinding describes the method used by a
// MiddlewareAdapter to server dynamically.
type middlewareBinding struct {
	caller      miruken.CallerFunc
	argIndexes  []int
}


// getMiddlewareMethod discovers a suitable Middleware method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getMiddlewareMethod(
	middleware Middleware,
) (*middlewareBinding, error) {
	typ := reflect.TypeOf(middleware)
	if bindings := middlewareBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
	}
	middlewareFuncLock.Lock()
	defer middlewareFuncLock.Unlock()
	bindings := middlewareBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
		sb := maps.Clone(*bindings)
		bindings = &sb
	} else {
		bindings = &map[reflect.Type]middlewareBinding{}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name == "ServeHTTP" {
			continue
		}
		if lateServeType := method.Type;
			lateServeType.NumIn() < 4 || lateServeType.NumOut() > 0 {
			continue
		} else {
			binding := middlewareBinding{}
			numArgs := lateServeType.NumIn()
			for i := 1; i < 3; i++ {
				if lateServeType.In(i) != middlewareFuncType.In(i) {
					continue
				}
			}
			for i := 3; i < numArgs; i++ {
				for j := 3; j < middlewareFuncType.NumIn(); j++ {
					if lateServeType.In(i) == middlewareFuncType.In(j) {
						binding.argIndexes = append(binding.argIndexes, j)
						break
					}
				}
			}
			caller, err := miruken.MakeCaller(method.Func)
			if err != nil {
				return nil, &miruken.MethodBindingError{Method: method, Cause: err}
			}
			binding.caller   = caller
			(*bindings)[typ] = binding
			middlewareBindingMap.Store(bindings)
			return &binding, nil
		}
	}
	return nil, fmt.Errorf(`middleware: %v has no compatible dynamic method`, typ)
}


type (
	// baseAdapter provides common behavior for adapting a pipeline.
	// to various http handler mechanisms.
	baseAdapter struct {
		ctx      *context.Context
		pipeline Middleware
	}

	// rawAdapter adapts a standard http.Handler to a pipeline.
	rawAdapter struct {
		baseAdapter
		handler  http.Handler
	}

	// useAdapter adapts an enhanced Handler to a pipeline.
	useAdapter struct {
		baseAdapter
		handler  Handler
	}

	// funAdapter adapts a handler function to a pipeline.
	funAdapter struct {
		baseAdapter
		handler miruken.CallerFunc
	}

	// resolveAdapter adapts a resolved Handler to a pipeline.
	resolveAdapter[H Handler] struct {
		baseAdapter
	}

	// Specialized key type for context values.
	contextKey int
)

// ComposerKey is used to access the miruken.Handler from the context.
const ComposerKey contextKey = 0


func (a *baseAdapter) serve(
	w http.ResponseWriter,
	r *http.Request,
	h func(c miruken.Handler),
) {
	defer handlePanic(w, r)
	child := a.ctx.NewChild()
	defer child.Dispose()

	a.pipeline.ServeHTTP(a.pipeline, w, r, nil, child, func(c miruken.Handler) {
		if c == nil {
			c = child
		}
		h(c)
	})
}

func (a *rawAdapter) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		ctxWithComposer := context2.WithValue(r.Context(), ComposerKey, c)
		a.handler.ServeHTTP(w, r.WithContext(ctxWithComposer))
	})
}

func (a *useAdapter) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		a.handler.ServeHTTP(w, r, c)
	})
}

func (a *funAdapter) tryBind(fun any) bool {
	typ := reflect.TypeOf(fun)
	if typ.Kind() != reflect.Func {
		return false
	}
	if typ.NumIn() < 2 || typ.NumOut() > 0 {
		return false
	}
	if typ.In(0) != middlewareFuncType.In(1) ||
		typ.In(1) != middlewareFuncType.In(2) {
		return false
	}
	handler, err := miruken.MakeCaller(fun)
	if err != nil {
		return false
	}
	a.handler = handler
	return true
}

func (a *funAdapter) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		initArgs := []any{w,r}
		_, pr, err := a.handler(c, initArgs...)
		if err != nil && pr != nil {
			_, err = pr.Await()
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}

func (a *resolveAdapter[H]) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.serve(w, r, func(c miruken.Handler) {
		handler, ph, ok, err := provides.Type[H](c)
		if !(ok && err == nil) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if ph != nil {
			if handler, err = ph.Await(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		handler.ServeHTTP(w, r, c)
	})
}


// Use builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating
// at a provided handler.
func Use(
	composer miruken.Handler,
	handler    any,
	middleware ...Middleware,
) http.Handler {
	ctx, ok := composer.(*context.Context)
	if !ok {
		ctx = context.New(composer)
	}

	pipeline := Pipe(middleware...)

	switch h := handler.(type) {
	case http.Handler:
		return &rawAdapter{baseAdapter{ctx, pipeline}, h}
	case Handler:
		return &useAdapter{baseAdapter{ctx, pipeline}, h}
	case func(http.ResponseWriter, *http.Request):
		return &rawAdapter{baseAdapter{ctx, pipeline}, http.HandlerFunc(h)}
	case func(http.ResponseWriter, *http.Request,  miruken.Handler):
		return &useAdapter{baseAdapter{ctx, pipeline}, HandlerFunc(h)}
	default:
		fun := &funAdapter{baseAdapter: baseAdapter{ctx, pipeline}}
		if fun.tryBind(handler) {
			return fun
		}
		panic(fmt.Errorf(
			"httpsrv: %T is not a http.Handler, httpsrv.Handler or compatible handler function",
			handler))
	}
}

// Resolve builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating
// at a resolved Handler.
func Resolve[H Handler](
	composer   miruken.Handler,
	middleware ...Middleware,
) http.Handler {
	ctx, ok := composer.(*context.Context)
	if !ok {
		ctx = context.New(composer)
	}
	pipeline := Pipe(middleware...)
	return &resolveAdapter[H]{baseAdapter{ctx, pipeline}}
}

// Api builds a http.Handler for accepting polymorphic api calls
// through a Middleware pipeline.
func Api(
	composer   miruken.Handler,
	middleware ...Middleware,
) http.Handler {
	return Resolve[*ApiHandler](composer, middleware...)
}

// Pipe builds a Middleware chain for pre and post processing of requests.
func Pipe(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(
		s Middleware,
		w http.ResponseWriter,
		r *http.Request,
		m Middleware,
		h miruken.Handler,
		n func(miruken.Handler),
	) {
		index, length := 0, len(middleware)
		var next func(miruken.Handler)
		next = func(composer miruken.Handler) {
			if composer == nil {
				composer = h
			}
			if index < length {
				m := middleware[index]
				index++
				mm, pm, _, err := provides.Key[Middleware](composer, reflect.TypeOf(m))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else if pm != nil {
					if mm, err = pm.Await(); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
				if mm == nil {
					mm = m
				}
				mm.ServeHTTP(mm, w, r, m, composer, next);
			} else {
				n(composer)
			}
		}
		next(h)
	})
}


func handlePanic(w http.ResponseWriter, _ *http.Request) {
	if rc := recover(); rc != nil {
		buf := make([]byte, 2048)
		n   := runtime.Stack(buf, false)
		buf = buf[:n]
		log.Printf("recovering from http panic: %v\n%s", rc, string(buf))
		w.WriteHeader(http.StatusInternalServerError)
	}
}


var (
	middlewareFuncLock sync.Mutex
	middlewareFuncType   = internal.TypeOf[MiddlewareFunc]()
	middlewareBindingMap = atomic.Pointer[map[reflect.Type]middlewareBinding]{}
)
