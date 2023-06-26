package httpsrv

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
	"net/http"
	"reflect"
)

type (
	// Middleware extends http.Handler to include a request scoped context.
	Middleware interface {
		ServeHTTP(
			w http.ResponseWriter,
			r *http.Request,
			h miruken.Handler,
			m Middleware,
			n func(handler miruken.Handler),
		)
	}

	// MiddlewareFunc promotes a function to Middleware.
	MiddlewareFunc func(
		w http.ResponseWriter,
		r *http.Request,
		h miruken.Handler,
		m Middleware,
		n func(handler miruken.Handler),
	)
)


func (f MiddlewareFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	m Middleware,
	n func(handler miruken.Handler),
) { f(w, r, h, m, n) }



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
				mm.ServeHTTP(w, r, h, m, next)
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
