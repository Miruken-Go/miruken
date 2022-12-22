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
	"reflect"
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
			var err error
			if code == http.StatusUnsupportedMediaType {
				err = &miruken.NotHandledError{Callback: routed}
			} else {
				err = r.decodeError(res, format, composer)
			}
			if err == nil {
				reject(errors.New(res.Status))
			} else {
				reject(fmt.Errorf("http router: (%s) %w", res.Status, err))
			}
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
	format   string,
	composer miruken.Handler,
) error {
	contentType := res.Header.Get("Content-Type")
	if len(contentType) == 0 {
		contentType = format
	}
	if msg, _, err := miruken.Map[api.Message](composer, res.Body, miruken.From(format)); err == nil {
		if payload := msg.Payload; payload != nil {
			if err, _, ae := miruken.Map[error](composer, payload, api.ToError); ae == nil {
				return err
			} else {
				// If mapping failed and error payload is a slice, attempt to coerce
				// the slice into the most specific element type.  This is necessary
				// since polymorphic slices will be decoded as []any and mapping
				// functions typically declare concrete slices.
				var nh *miruken.NotHandledError
				if errors.As(ae, &nh) {
					if val := reflect.ValueOf(payload); val.Type().Kind() == reflect.Slice {
						if sv, ok := miruken.CoerceSlice(val, nil); ok {
							slice := sv.Interface()
							if err, _, ae = miruken.Map[error](composer, slice, api.ToError); ae == nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
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
