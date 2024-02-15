package httpsrv

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Middleware augments Handler to participate in a pipeline to support
	// pre and post processing of requests.
	Middleware interface {
		ServeHTTP(
			http.ResponseWriter,
			*http.Request,
			miruken.Handler,
			func(miruken.Handler),
		)
	}

	// MiddlewareFunc promotes a function to Middleware.
	MiddlewareFunc func(
		w http.ResponseWriter,
		r *http.Request,
		h miruken.Handler,
		n func(miruken.Handler),
	)
)

func (f MiddlewareFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	f(w, r, h, n)
}

// Pipe builds a Middleware chain for pre and post processing of http requests.
func Pipe(middleware ...any) Middleware {
	ms := slices.Map[any, Middleware](middleware, func(m any) Middleware {
		switch mm := m.(type) {
		case Middleware:
			return mm
		case func(http.ResponseWriter, *http.Request, miruken.Handler, func(miruken.Handler)):
			return MiddlewareFunc(mm)
		default:
			fun := &funMiddleware{}
			if fun.tryBind(m) {
				return fun
			}
			panic(fmt.Errorf(
				"httpsrv: %T is not httpsrv.Middleware or compatible middleware function", m))
		}
	})
	return MiddlewareFunc(func(
		w http.ResponseWriter,
		r *http.Request,
		h miruken.Handler,
		n func(miruken.Handler),
	) {
		index, length := 0, len(ms)
		var next func(miruken.Handler)
		next = func(composer miruken.Handler) {
			if composer == nil {
				composer = h
			}
			if index < length {
				m := ms[index]
				index++
				m.ServeHTTP(w, r, composer, next)
			} else {
				n(composer)
			}
		}
		next(h)
	})
}

// M builds a typed Middleware wrapper for dynamic http processing.
func M[M any](opts ...any) Middleware {
	typ := reflect.TypeFor[M]()
	if typ.Implements(middlewareType) {
		return &resMiddleware{typ: typ, opts: opts}
	}
	if binding, err := getMiddlewareBinding(typ); binding != nil && err == nil {
		return &dynMiddleware{typ: typ, binding: *binding, opts: opts}
	}
	panic(fmt.Errorf(
		"httpsrv: %v is not httpsrv.Middleware or compatible middleware type", typ))
}

type (
	// funMiddleware adapts a function to a pipeline.
	funMiddleware struct {
		binding middlewareBinding
	}

	// resMiddleware adapts resolved Middleware to a pipeline.
	resMiddleware struct {
		typ  reflect.Type
		opts []any
	}

	// dynMiddleware adapts resolved compatible middleware to a pipeline.
	dynMiddleware struct {
		typ     reflect.Type
		binding middlewareBinding
		opts    []any
	}

	// middlewareBinding represents the method used by
	// dynamic Middleware to serve http content.
	middlewareBinding struct {
		caller     miruken.CallerFunc
		argIndexes []int
	}
)

// getMiddlewareBinding discovers a suitable Middleware method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getMiddlewareBinding(
	typ reflect.Type,
) (*middlewareBinding, error) {
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
	for i := range typ.NumMethod() {
		method := typ.Method(i)
		binding, err := makeMiddlewareBinding(method.Type, method.Func, 1)
		if binding != nil {
			(*bindings)[typ] = *binding
			middlewareBindingMap.Store(bindings)
			return binding, nil
		} else if err != nil {
			return nil, &miruken.MethodBindingError{Method: &method, Cause: err}
		}
	}
	return nil, fmt.Errorf(`httpsrv: middleware %v has no compatible dynamic method`, typ)
}

func makeMiddlewareBinding(
	typ reflect.Type,
	fun reflect.Value,
	idx int,
) (*middlewareBinding, error) {
	if typ.NumIn() < 2 || typ.NumOut() > 0 {
		return nil, nil
	}
	binding := middlewareBinding{}
	numArgs := typ.NumIn()
	if typ.In(idx) != middlewareFuncType.In(0) ||
		typ.In(idx+1) != middlewareFuncType.In(1) {
		return nil, nil
	}
	for i := 2 + idx; i < numArgs; i++ {
		for j := 2; j < middlewareFuncType.NumIn(); j++ {
			if typ.In(i) == middlewareFuncType.In(j) {
				binding.argIndexes = append(binding.argIndexes, j)
				break
			}
		}
	}
	if caller, err := miruken.MakeCaller(fun); err != nil {
		return nil, err
	} else {
		binding.caller = caller
		return &binding, nil
	}
}

func (a *funMiddleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	if err := a.binding.invoke(h, n, w, r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *funMiddleware) tryBind(fun any) bool {
	typ := reflect.TypeOf(fun)
	if typ.Kind() != reflect.Func {
		return false
	}
	binding, err := makeMiddlewareBinding(typ, reflect.ValueOf(fun), 0)
	if binding != nil && err == nil {
		a.binding = *binding
		return true
	}
	return false
}

func (a *resMiddleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	if opts := a.opts; len(opts) > 0 {
		h = miruken.BuildUp(h, provides.With(opts...))
	}
	middleware, pm, ok, err := miruken.ResolveKey[Middleware](h, a.typ)
	if !(ok && err == nil) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if pm != nil {
		if middleware, err = pm.Await(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	middleware.ServeHTTP(w, r, h, n)
}

func (a *dynMiddleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	if opts := a.opts; len(opts) > 0 {
		h = miruken.BuildUp(h, provides.With(opts...))
	}
	middleware, pm, ok, err := miruken.ResolveKey[any](h, a.typ)
	if !(ok && err == nil) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if pm != nil {
		if middleware, err = pm.Await(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if err := a.binding.invoke(h, n, middleware, w, r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (b middlewareBinding) invoke(
	h miruken.Handler,
	n func(miruken.Handler),
	initArgs ...any,
) error {
	for _, idx := range b.argIndexes {
		switch idx {
		case 2:
			initArgs = append(initArgs, h)
		case 3:
			initArgs = append(initArgs, n)
		}
	}
	_, pr, err := b.caller(h, initArgs...)
	if err != nil && pr != nil {
		_, err = pr.Await()
	}
	return err
}

func handlePanic(w http.ResponseWriter, _ *http.Request) {
	if rc := recover(); rc != nil {
		buf := make([]byte, 2048)
		n := runtime.Stack(buf, false)
		buf = buf[:n]
		log.Printf("recovering from http panic: %v\n%s", rc, string(buf))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

var (
	middlewareType = reflect.TypeFor[Middleware]()

	middlewareFuncLock   sync.Mutex
	middlewareFuncType   = reflect.TypeFor[MiddlewareFunc]()
	middlewareBindingMap = atomic.Pointer[map[reflect.Type]middlewareBinding]{}
)
