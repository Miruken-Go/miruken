package httpsrv

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
	"net/http"
	"strings"
)

type Controller struct {
	ctx *miruken.Context
}

func (c *Controller) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	valid, format, publish := c.validateRequest(w, r)
	if !valid {
		return
	}
	child := c.ctx.NewChild()
	defer child.Dispose()
	if msg, _, err := miruken.Map[api.Message](child, r.Body, miruken.From(format)); err != nil {
		c.encodeError(err, format, w, child)
	} else if msg.Payload == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if publish {
		if pv, err := api.Publish(child, msg.Payload); err != nil {
			c.encodeError(err, format, w, child)
		} else if pv == nil {
			c.encodeResult(nil, format, w, child)
		} else if _, err = pv.Await(); err == nil {
			c.encodeResult(nil, format, w, child)
		} else {
			c.encodeError(err, format, w, child)
		}
	} else if res, pr, err := api.Send[any](child, msg.Payload); err != nil {
		c.encodeError(err, format, w, child)
	} else if pr == nil {
		c.encodeResult(res, format, w, child)
	} else if res, err = pr.Await(); err == nil {
		c.encodeResult(res, format, w, child)
	} else {
		c.encodeError(err, format, w, child)
	}
}

func (c *Controller) validateRequest(
	w http.ResponseWriter,
	r *http.Request,
) (valid bool, format string, publish bool) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return false, "",false
	}
	format = r.Header.Get("Content-Type")
	if len(format) == 0 {
		http.Error(w, "400 missing 'Content-Type' header", http.StatusBadRequest)
		return false, "", false
	}
	if path := r.RequestURI;
		path == "/process" || strings.HasPrefix(path, "/process/") {
		return true, format,false
	} else if path == "/publish" || strings.HasPrefix(path, "/publish/") {
		return true, format, true
	}
	http.Error(w, "404 not found", http.StatusNotFound)
	return false, format,false
}

func (c *Controller) encodeResult(
	res    any,
	format string,
	w      http.ResponseWriter,
	ctx    *miruken.Context,
) {
	w.Header().Set("Content-Type", format)
	out := io.Writer(w)
	msg := api.Message{Payload: res}
	if _, err := miruken.MapInto(ctx, msg, &out, miruken.To(format)); err != nil {
		c.encodeError(err, format, w, ctx)
	}
}

func (c *Controller) encodeError(
	err    error,
	format string,
	w      http.ResponseWriter,
	ctx    *miruken.Context,
) {
	w.Header().Set("Content-Type", format)
	statusCode := http.StatusInternalServerError
	handler    := miruken.BuildUp(ctx, miruken.BestEffort)
	if sc, _, sce := miruken.Map[int](handler, err, toStatusCode); sc != 0 && sce == nil {
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

func NewController(ctx *miruken.Context) *Controller {
	if ctx == nil {
		panic("ctx cannot be nil")
	}
	return &Controller{ctx}
}

var toStatusCode = miruken.To("http:status-code")
