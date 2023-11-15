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
			required := 0
			for i := 3; i < numArgs; i++ {
				for j := 3; j < middlewareFuncType.NumIn(); j++ {
					if lateServeType.In(i) == middlewareFuncType.In(j) {
						binding.argIndexes = append(binding.argIndexes, j)
						if j == 4 || j == 5 {
							required++
						}
						break
					}
				}
			}
			if required != 2 {
				continue
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

		pipeline.ServeHTTP(pipeline, w, r, nil, child, func(c miruken.Handler) {
			if c == nil {
				c = child
			}
			handler.ServeHTTP(w, r, c)
		})
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

		pipeline.ServeHTTP(pipeline, w, r, nil, child, func(c miruken.Handler) {
			if c == nil {
				c = child
			}
			hh.ServeHTTP(w, r, c)
		})
	})
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
