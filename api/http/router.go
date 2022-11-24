package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"net/http"
	"net/url"
	"time"
)

type (
	// Options customizes http operations.
	Options struct {
		Timeout time.Duration
	}

	// Router routes messages over a http transport.
	Router struct {}
)

func (r *Router) Route(
	_*struct{
		miruken.Handles
		miruken.Singleton
		api.Routes `scheme:"http,https"`
	  }, routed api.Routed,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return promise.New(func(resolve func(any), reject func(error)) {
		uri, err := r.getResourceUri(routed, &ctx)
		if err != nil {
			reject(err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, uri, nil)
		if err != nil {
			reject(err)
			return
		}
		client := http.Client{
			Timeout: 30 * time.Second,
		}
		if _, err := client.Do(req); err != nil {
			reject(err)
		}
	})
}

func (r *Router) getResourceUri(
	routed  api.Routed,
	ctx     *miruken.HandleContext,
) (string, error) {
	if ctx.Greedy() {
		return url.JoinPath(routed.Route, "publish")
	}
	return url.JoinPath(routed.Route, "process")
}