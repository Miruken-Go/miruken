package http

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"io"
	"net/http"
	"net/url"
	"time"
)

type (
	// Options customize http operations.
	Options struct {
		Timeout miruken.Option[time.Duration]
		Format  miruken.Option[string]
	}

	// Router routes messages over a http transport.
	Router struct {}
)

const defaultTimeout     = 30 * time.Second
const defaultContentType = "application/json"

func (r *Router) Route(
	_*struct{
		miruken.Handles
		miruken.Singleton
		api.Routes `scheme:"http,https"`
	  }, routed api.Routed,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return promise.New(func(resolve func(any), reject func(error)) {
		uri, err := r.getResourceUri(routed, &ctx)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		composer := ctx.Composer()
		format   := options.Format.ValueOrDefault(defaultContentType)

		var b bytes.Buffer
		out := io.Writer(&b)
		msg := api.Message{Payload: routed.Message}
		if _, err = miruken.MapInto(composer, msg, &out, miruken.To(format)); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		}

		req, err  := http.NewRequest(http.MethodPost, uri, &b)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}
		req.Header.Add("Content-Type", format)

		client := &http.Client{Timeout: options.Timeout.ValueOrDefault(defaultTimeout)}
		res, err := client.Do(req)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		if code := res.StatusCode; code < 200 || code >= 300 {
			err := r.decodeError(res, composer)
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		contentType := res.Header.Get("Content-type")
		if len(contentType) == 0 {
			contentType = format
		}
		if msg, _, err := miruken.Map[api.Message](composer, res.Body, miruken.From(format)); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		} else {
			resolve(msg.Payload)
		}
	})
}

func (r *Router) decodeError(
	res      *http.Response,
	composer miruken.Handler,
) error {
	return errors.New(res.Status)
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
