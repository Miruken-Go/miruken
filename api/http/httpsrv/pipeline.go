package httpsrv

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/provides"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

type (
	// Middleware augments http.Handler to provide pre and post
	// processing of requests and responses.
	Middleware interface {
		ServeHTTP(
			w http.ResponseWriter,
			r *http.Request,
			m Middleware,
			h miruken.Handler,
			n func(miruken.Handler),
		) error
	}

	// MiddlewareFunc promotes a function to Middleware.
	MiddlewareFunc func(
		w http.ResponseWriter,
		r *http.Request,
		m Middleware,
		h miruken.Handler,
		n func(miruken.Handler),
	) error
)


func (f MiddlewareFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	m Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) error { return f(w, r, m, h, n) }


func ServeHTTPLate(
	m Middleware,
	w http.ResponseWriter,
	r *http.Request,
	p Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) error {
	if binding, err := getServeHTTPLate(reflect.TypeOf(m)); err != nil {
		return err
	} else {
		_, pr, err := binding(h, m, w, r, p, h, n)
		if pr != nil {
			_, err = pr.Await()
		}
		return err
	}
}


// getServeHTTPLate discovers a suitable "ServeHTTPLate" method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getServeHTTPLate(typ reflect.Type) (miruken.CallerFunc, error) {
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
		cb := make(map[reflect.Type]miruken.CallerFunc, len(*callers)+1)
		for k, v := range *callers {
			cb[k] = v
		}
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
	return nil, fmt.Errorf(
		`middleware: %v missing valid dynamic "ServeHTTP" method`, typ)
}


// Pipeline returns a http.Handler for processing api calls
// through a list of Middleware components.
func Pipeline(
	handler    miruken.Handler,
	middleware ...Middleware,
) http.Handler {
	ctx, ok := handler.(*context.Context)
	if !ok {
		ctx = context.New(handler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer handlePanic(w, r)

		child := ctx.NewChild()
		defer child.Dispose()

		index, length := 0, len(middleware)
		var next func(miruken.Handler)
		next = func(h miruken.Handler) {
			if h == nil {
				h = child
			}
			if index < length {
				m := middleware[index]
				index++
				mm, mp, err := provides.Key[Middleware](h, reflect.TypeOf(m))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else if mp != nil {
					if mm, err = mp.Await(); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
				if mm == nil {
					mm = m
				}
				if err := mm.ServeHTTP(w, r, m, h, next); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				a, cp, err := provides.Type[*ApiHandler](h)
				if a == nil || err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else if cp != nil {
					if a, err = cp.Await(); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
				a.ServeHTTP(w, r, h)
			}
		}

		next(child)
	})
}

func handlePanic(w http.ResponseWriter, r *http.Request) {
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
