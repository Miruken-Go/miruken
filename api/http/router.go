package http

import (
	"bytes"
	"encoding/json"
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

	// Message is an envelope for polymorphic payloads.
	Message struct {
		Payload *json.RawMessage `json:"payload"`
	}

	// Router routes messages over a http transport.
	Router struct {}
)

const defaultTimeout = 30 * time.Second

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
		format   := options.Format.ValueOrDefault(_defaultContentType)
		pay, err := r.encodePayload(routed.Message, format, composer)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
		}

		var body bytes.Buffer
		enc := json.NewEncoder(&body)
		msg := Message{(*json.RawMessage)(&pay)}
		if err := enc.Encode(msg); err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		req, err  := http.NewRequest(http.MethodPost, uri, &body)
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

		if r, err := r.decodeResponse(res, composer); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		} else {
			resolve(r)
		}
	})
}

func (r *Router) encodePayload(
	msg      any,
	format   string,
	composer miruken.Handler,
) ([]byte, error) {
	var payload bytes.Buffer
	stream := io.Writer(&payload)
	if _, err := miruken.MapInto(
		miruken.BuildUp(composer, _polyOptions),
		msg, &stream, &miruken.Format{As: format}); err == nil {
		return payload.Bytes(), nil
	} else {
		return nil, err
	}
}

func (r *Router) decodeResponse(
	res      *http.Response,
	composer miruken.Handler,
) (any, error) {
	var msg Message
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	} else if pay := msg.Payload; pay == nil {
		return nil, nil
	} else {
		contentType := res.Header.Get("Content-type")
		if len(contentType) == 0 {
			contentType = _defaultContentType
		}
		data, _, err := miruken.Map[any](
			miruken.BuildUp(composer, _polyOptions),
			bytes.NewReader(*pay), &miruken.Format{As: contentType})
		return data, err
	}
}

func (r *Router) decodeError(
	res      *http.Response,
	composer miruken.Handler,
) error {
	return fmt.Errorf("http router: %s (%d)", res.Status, res.StatusCode)
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

var (
	_defaultContentType = "application/json"

	_polyOptions = miruken.Options(api.PolymorphicOptions{
		PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot),
	})
)