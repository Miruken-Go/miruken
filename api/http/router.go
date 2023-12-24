package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Policy defines custom behavior for http requests.
	Policy interface {
		Apply(
			req      *http.Request,
			composer miruken.Handler,
			next     func() (*http.Response, error),
		) (*http.Response, error)
	}

	// PolicyFunc promotes a function to Policy.
	PolicyFunc func(
		req      *http.Request,
		composer miruken.Handler,
		next     func() (*http.Response, error),
	) (*http.Response, error)

	// Options customize http operations.
	Options struct {
		Format      string
		ProcessPath string
		PublishPath string
		Pipeline    []Policy
	}

	// Router routes messages over a http transport.
	Router struct{}
)

const (
	defaultFormat  = "application/json"
	defaultTimeout = 30 * time.Second
)

func (f PolicyFunc) Apply(
	req      *http.Request,
	composer miruken.Handler,
	next     func() (*http.Response, error),
) (*http.Response, error) {
	return f(req, composer, next)
}

func (r *Router) Route(
	_ *struct {
		handles.It
		api.Routes `scheme:"http,https"`
	  }, routed api.Routed,
	_ *struct {
		args.Optional
		args.FromOptions
	  }, options Options,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		uri, err := r.resourceUri(routed, &options, &ctx)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		var format string
		if format = options.Format; format == "" {
			format = defaultFormat
		}
		to, err := api.ParseMediaType(format, maps.DirectionTo)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}

		composer := miruken.BuildUp(ctx.Composer, api.Polymorphic)

		var b bytes.Buffer
		out := io.Writer(&b)
		msg := api.Message{Payload: routed.Message}
		if _, _, err = maps.Into(composer, msg, &out, to); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		}

		req, err := http.NewRequest(http.MethodPost, uri, &b)
		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}
		req.Header.Add("Content-Type", format)

		res, err := r.invoke(req, composer, options.Pipeline)

		if err != nil {
			reject(fmt.Errorf("http router: %w", err))
			return
		}
		defer func(body io.ReadCloser) {
			_ = body.Close()
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
		if contentType == "" {
			contentType = format
		}
		if from, err := api.ParseMediaType(contentType, maps.DirectionFrom); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		} else if msg, _, _, err := maps.Out[api.Message](composer, res.Body, from); err != nil {
			reject(fmt.Errorf("http router: %w", err))
		} else {
			resolve(msg.Payload)
		}
	})
}

func (r *Router) invoke(
	req      *http.Request,
	composer miruken.Handler,
	pipeline []Policy,
) (*http.Response, error) {
	index, length := 0, len(pipeline)
	if length == 0 {
		return defaultHttpClient.Do(req)
	}
	var next func() (*http.Response, error)
	next = func() (*http.Response, error) {
		if index < length {
			policy := pipeline[index]
			index++
			return policy.Apply(req, composer, next)
		}
		return defaultHttpClient.Do(req)
	}
	return next()
}

func (r *Router) decodeError(
	res      *http.Response,
	format   string,
	composer miruken.Handler,
) error {
	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = format
	}
	from, err := api.ParseMediaType(contentType, maps.DirectionFrom)
	if err != nil {
		return err
	}
	msg, _, _, err := maps.Out[api.Message](composer, res.Body, from)
	if err == nil {
		if payload := msg.Payload; payload != nil {
			if err, ok := payload.(error); ok {
				return err
			} else {
				return &api.MalformedErrorError{Culprit: payload}
			}
		}
	}
	return nil
}

func (r *Router) resourceUri(
	routed   api.Routed,
	options *Options,
	ctx     *miruken.HandleContext,
) (string, error) {
	var path string
	if ctx.Greedy {
		if path = options.PublishPath; path == "" {
			path = "publish"
		}
	} else if path = options.ProcessPath; path == "" {
		path = "process"
	}
	return url.JoinPath(routed.Route, path)
}

// Client returns a Policy to use the supplied http.Client.
// This should be the last Policy in the pipeline.
func Client(client *http.Client) Policy {
	return PolicyFunc(func(
		req *http.Request,
		_   miruken.Handler,
		_   func() (*http.Response, error),
	) (*http.Response, error) {
		return client.Do(req)
	})
}

// Format returns a miruken.Builder requesting a specific format.
func Format(format string) miruken.Builder {
	return miruken.Options(Options{Format: format})
}

// Pipeline returns a miruken.Builder that registers policies to
// apply during http request processing.
func Pipeline(policies ...Policy) miruken.Builder {
	return miruken.Options(Options{Pipeline: policies})
}

// newDefaultHttpClient creates an optimized http.Client
// https://www.loginradius.com/blog/engineering/tune-the-go-http-client-for-high-performance/
func newDefaultHttpClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	return &http.Client{
		Timeout:   defaultTimeout,
		Transport: t,
	}
}

var defaultHttpClient = newDefaultHttpClient()
