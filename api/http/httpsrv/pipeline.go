package httpsrv

import (
	context2 "context"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
	"google.golang.org/appengine/log"
	"net/http"
	"reflect"
	"runtime"
	"sync"
)

type (
	// Middleware extends http.Handler to include a request scoped context.
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


func DynServeHTTP(
	m Middleware,
	w http.ResponseWriter,
	r *http.Request,
	p Middleware,
	h miruken.Handler,
	n func(miruken.Handler),
) error {
	if binding, err := getServeHTTPBinding(reflect.TypeOf(m)); err != nil {
		return err
	} else {
		_, pr, err := binding(h, m, w, r, p, h, n)
		if pr != nil {
			_, err = pr.Await()
		}
		return err
	}
}

func getServeHTTPBinding(typ reflect.Type) (miruken.CallerFunc, error) {
	dynMiddlewareLock.RLock()
	binding := dynMiddlewareMap[typ]
	dynMiddlewareLock.RUnlock()
	if binding == nil {
		dynMiddlewareLock.Lock()
		defer dynMiddlewareLock.Unlock()
		if binding = dynMiddlewareMap[typ]; binding == nil {
			if dynServeHTTP, ok := typ.MethodByName("DynServeHTTP"); !ok {
				goto Invalid
			} else if dynNextType := dynServeHTTP.Type;
				dynNextType.NumIn() < middlewareFuncType.NumIn() ||
					dynNextType.NumOut() < middlewareFuncType.NumOut() {
				goto Invalid
			} else {
				for i := 0; i < middlewareFuncType.NumIn(); i++ {
					if dynNextType.In(i+1) != middlewareFuncType.In(i) {
						goto Invalid
					}
				}
				for i := 0; i < middlewareFuncType.NumOut(); i++ {
					if dynNextType.Out(i) != middlewareFuncType.Out(i) {
						goto Invalid
					}
				}
				caller, err := miruken.MakeCaller(dynServeHTTP.Func)
				if err != nil {
					err = fmt.Errorf("DynServeHTTP: %w", err)
					return nil, &miruken.MethodBindingError{Method: dynServeHTTP, Cause: err}
				}
				dynMiddlewareMap[typ] = caller
				binding = caller
			}
		}
	}
	return binding, nil
Invalid:
	return nil, fmt.Errorf(
		"middleware %v missing valid DynServeHTTP(...) method", typ)
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
		defer handlePanic(w)

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

func handlePanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		buf := make([]byte, 2048)
		n := runtime.Stack(buf, false)
		buf = buf[:n]
		log.Errorf(context2.Background(),"recovering from http panic: %v\n%s", r, string(buf))
		w.WriteHeader(http.StatusInternalServerError)
	}
}


var (
	dynMiddlewareLock sync.RWMutex
	middlewareFuncType = miruken.TypeOf[MiddlewareFunc]()
	dynMiddlewareMap   = make(map[reflect.Type]miruken.CallerFunc)
)
