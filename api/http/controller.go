package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
	"net/http"
	"strings"
)

type (
	Controller struct {
		Context *miruken.Context
	}
)

func (c *Controller) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	valid, publish := c.validateRequest(w, r)
	if !valid {
		return
	}
	ctx := c.Context.NewChild()
	defer ctx.Dispose()
	format := r.Header.Get("Content-Type")
	if len(format) == 0 {
		format = defaultContentType
	}
	if msg, _, err := miruken.Map[api.Message](ctx, r.Body, miruken.From(format)); err != nil {
		c.encodeError(err, format, w, ctx)
	} else if msg.Payload == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if publish {
		if pv, err := api.Publish(ctx, msg.Payload); err != nil {
			c.encodeError(err, format, w, ctx)
		} else if pv == nil {
			c.encodeResult(nil, format, w, ctx)
		} else if _, err = pv.Await(); err == nil {
			c.encodeResult(nil, format, w, ctx)
		} else {
			c.encodeError(err, format, w, ctx)
		}
	} else if res, pr, err := api.Send[any](ctx, msg.Payload); err != nil {
		c.encodeError(err, format, w, ctx)
	} else if pr == nil {
		c.encodeResult(res, format, w, ctx)
	} else if res, err = pr.Await(); err == nil {
		c.encodeResult(res, format, w, ctx)
	} else {
		c.encodeError(err, format, w, ctx)
	}
}

func (c *Controller) validateRequest(
	w http.ResponseWriter,
	r *http.Request,
) (valid bool, publish bool) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return false, false
	}
	if path := r.RequestURI;
		path == "/process" || strings.HasPrefix(path, "/process/") {
		return true, false
	} else if path == "/publish" || strings.HasPrefix(path, "/publish/") {
		return true, true
	}
	http.Error(w, "404 not found", http.StatusNotFound)
	return false, false
}

func (c *Controller) encodeResult(
	res      any,
	format   string,
	w        http.ResponseWriter,
	ctx      *miruken.Context,
) {
	w.Header().Set("Content-Type", format)
	out := io.Writer(w)
	msg := api.Message{Payload: res}
	if _, err := miruken.MapInto(ctx, msg, &out, miruken.To(format)); err != nil {
		c.encodeError(err, format, w, ctx)
	}
}

func (c *Controller) encodeError(
	err     error,
	format  string,
	w       http.ResponseWriter,
	ctx     *miruken.Context,
) {
	w.Header().Set("Content-Type", format)
	statusCode := http.StatusInternalServerError
	handler    := miruken.BuildUp(ctx, miruken.BestEffort)
	if sc, _, sce := miruken.Map[int](handler, err, _toStatusCode); sc != 0 && sce == nil {
		statusCode = sc
	}
	w.WriteHeader(statusCode)
	ap, _, ae := miruken.Map[any](handler, err, api.FromError)
	if miruken.IsNil(ap) || ae != nil {
		ap = api.ErrorData{Message: err.Error()}
	}
	out := io.Writer(w)
	msg := api.Message{Payload: ap}
	_, _ = miruken.MapInto(ctx, msg, &out, miruken.To(format))
}

var _toStatusCode = miruken.To("http:status-code")
