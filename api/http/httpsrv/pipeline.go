package httpsrv

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
	"net/http"
)

// Handler returns a http.Handler for processing api calls
// bound to the given miruken.Handler.
func Handler(handler miruken.Handler) http.Handler {
	if _, ok := handler.(*context.Context); !ok {
		handler = context.New(handler)
	}
	h, cp, err := provides.Type[*ApiHandler](handler)
	if err != nil {
		panic(err)
	}
	if cp != nil {
		if h, err = cp.Await(); err != nil {
			panic(err)
		}
	}
	return h
}

