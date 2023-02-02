package http

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

type (
	// Options customize http operations.
	Options struct {
		Format      string
		ProcessPath string
		PublishPath string
		Timeout     miruken.Option[time.Duration]
	}

	// Router routes messages over a http transport.
	Router struct {}
)

const defaultFormat = "application/json"
const defaultTimeout = 30 * time.Second

func (r *Router) Route(
	_*struct{
		handles.It
		provides.Single
		api.Routes `scheme:"http,https"`
	  }, routed api.Routed,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return promise.New(func(resolve func(any), reject func(error)) {
		uri, err := r.getResourceUri(routed, &options, &ctx)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		composer := ctx.Composer()

		var format string
		if format = options.Format; len(format) == 0 {
			format = defaultFormat
		}

		var b bytes.Buffer
		out := io.Writer(&b)
		msg := api.Message{Payload: routed.Message}
		if _, err = maps.MapInto(composer, msg, &out, maps.To(format)); err != nil {
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
		if msg, _, err := maps.Map[api.Message](composer, res.Body, maps.From(format)); err != nil {
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
	if msg, _, err := maps.Map[api.Message](composer, res.Body, maps.From(format)); err == nil {
		if payload := msg.Payload; payload != nil {
			if err, _, ae := maps.Map[error](composer, payload, api.ToError); ae == nil {
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
							if err, _, ae = maps.Map[error](composer, slice, api.ToError); ae == nil {
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
	options *Options,
	ctx     *miruken.HandleContext,
) (string, error) {
	var path string
	if ctx.Greedy() {
		if path = options.PublishPath; len(path) == 0 {
			path = "publish"
		}
	} else if path = options.ProcessPath; len(path) == 0 {
		path = "process"
	}
	return url.JoinPath(routed.Route, path)
}


// Format returns a miruken.Builder requesting a specific format.
func Format(format string) miruken.Builder {
	return miruken.Options(Options{Format: format})
}
