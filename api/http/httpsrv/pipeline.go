package httpsrv

import (
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
			http.ResponseWriter,
			*http.Request,
			Middleware,
			miruken.Handler,
			Handler,
		) error
	}

	// MiddlewareFunc promotes a function to Middleware.
	MiddlewareFunc func(
		w http.ResponseWriter,
		r *http.Request,
		m Middleware,
		h miruken.Handler,
		n Handler,
	) error
)


func (f HandlerFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
) { f(w, r, h) }


func (f MiddlewareFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	m Middleware,
	h miruken.Handler,
	n Handler,
) error { return f(w, r, m, h, n) }


func ServeHTTPLate(
	m Middleware,
	w http.ResponseWriter,
	r *http.Request,
	p Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) error {
	if binding, err := getServeHTTPCaller(reflect.TypeOf(m)); err != nil {
		return err
	} else {
		_, pr, err := binding(h, m, w, r, p, h, n)
		if pr != nil {
			_, err = pr.Await()
		}
		return err
	}
}

// getServeHTTPCaller discovers a suitable dynamic "ServeHttp" method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getServeHTTPCaller(typ reflect.Type) (miruken.CallerFunc, error) {
	if callers := middlewareFuncMap.Load(); callers != nil {
		if caller, ok := (*callers)[typ]; ok {
			return caller, nil
		}
	}
	middlewareFuncLock.Lock()
	defer middlewareFuncLock.Unlock()
	callers := middlewareFuncMap.Load()
	if callers != nil {
		if caller, ok := (*callers)[typ]; ok {
			return caller, nil
		}
		cb := maps.Clone(*callers)
		callers = &cb
	} else {
		callers = &map[reflect.Type]miruken.CallerFunc{}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name == "ServeHTTP" {
			continue
		}
		if lateNextType := method.Type;
			lateNextType.NumIn() < middlewareFuncType.NumIn() ||
				lateNextType.NumOut() < middlewareFuncType.NumOut() {
			continue
		} else {
			for i := 0; i < middlewareFuncType.NumIn(); i++ {
				if lateNextType.In(i+1) != middlewareFuncType.In(i) {
					continue
				}
			}
			for i := 0; i < middlewareFuncType.NumOut(); i++ {
				if lateNextType.Out(i) != middlewareFuncType.Out(i) {
					continue
				}
			}
			caller, err := miruken.MakeCaller(method.Func)
			if err != nil {
				return nil, &miruken.MethodBindingError{Method: method, Cause: err}
			}
			(*callers)[typ] = caller
			middlewareFuncMap.Store(callers)
			return caller, nil
		}
	}
	return nil, fmt.Errorf(`middleware: %v has no compatible dynamic method`, typ)
}


// Api builds a http.Handler for accepting polymorphic api calls
// through a Middleware pipeline.
func Api(
	h          miruken.Handler,
	middleware ...Middleware,
) http.Handler {
	return DispatchTo[*ApiHandler](h, middleware...)
}

// Dispatch builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating
// at a Handler.
func Dispatch(
	h          miruken.Handler,
	handler    Handler,
	middleware ...Middleware,
) http.Handler {
	ctx, ok := h.(*context.Context)
	if !ok {
		ctx = context.New(h)
	}

	pipeline := Pipe(middleware...)

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		defer handlePanic(w, r)
		child := ctx.NewChild()
		defer child.Dispose()

		err := pipeline.ServeHTTP(w, r, nil, child, handler)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// DispatchFunc builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating at
// a handler function.
func DispatchFunc(
	h          miruken.Handler,
	handler    func(http.ResponseWriter, *http.Request, miruken.Handler),
	middleware ...Middleware,
) http.Handler {
	return Dispatch(h, HandlerFunc(handler), middleware...)
}

// DispatchTo builds an enhanced http.Handler for processing
// requests through a Middleware pipeline and terminating
// at a typed Handler.
func DispatchTo[H Handler](
	h          miruken.Handler,
	middleware ...Middleware,
) http.Handler {
	ctx, ok := h.(*context.Context)
	if !ok {
		ctx = context.New(h)
	}

	pipeline := Pipe(middleware...)

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		defer handlePanic(w, r)
		child := ctx.NewChild()
		defer child.Dispose()

		hh, ph, ok, err := provides.Type[H](child)
		if !(ok && err == nil) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if ph != nil {
			if hh, err = ph.Await(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		err = pipeline.ServeHTTP(w, r, nil, child, hh)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// Pipe builds a Middleware chain for pre and post processing of requests.
func Pipe(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(
		w http.ResponseWriter,
		r *http.Request,
		m Middleware,
		h miruken.Handler,
		n Handler,
	) error {
		index, length := 0, len(middleware)
		var next Handler
		next = HandlerFunc(func(
			w        http.ResponseWriter,
			r        *http.Request,
			composer miruken.Handler,
		) {
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
				if err := mm.ServeHTTP(w, r, m, composer, next); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				n.ServeHTTP(w, r, composer)
			}
		})
		next.ServeHTTP(w, r, h)
		return nil
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
	middlewareFuncType = internal.TypeOf[MiddlewareFunc]()
	middlewareFuncMap  = atomic.Pointer[map[reflect.Type]miruken.CallerFunc]{}
)
