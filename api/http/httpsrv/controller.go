package httpsrv

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/maps"
	"io"
	"net/http"
	"strings"
)

// ApiController is an http.Handler for api messages.
type ApiController struct {
	ctx *context.Context
}

func (c *ApiController) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	accepted, format, publish := c.acceptRequest(w, r)
	if !accepted {
		return
	}
	child := c.ctx.NewChild()
	defer child.Dispose()
	handler := miruken.BuildUp(child, api.Polymorphic)
	if msg, _, err := maps.Map[api.Message](handler, r.Body, maps.From(format)); err != nil {
		c.encodeError(err, format, w, handler)
	} else if msg.Payload == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if publish {
		if pv, err := api.Publish(handler, msg.Payload); err != nil {
			c.encodeError(err, format, w, handler)
		} else if pv == nil {
			c.encodeResult(nil, format, w, handler)
		} else if _, err = pv.Await(); err == nil {
			c.encodeResult(nil, format, w, handler)
		} else {
			c.encodeError(err, format, w, handler)
		}
	} else if res, pr, err := api.Send[any](handler, msg.Payload); err != nil {
		c.encodeError(err, format, w, handler)
	} else if pr == nil {
		c.encodeResult(res, format, w, handler)
	} else if res, err = pr.Await(); err == nil {
		c.encodeResult(res, format, w, handler)
	} else {
		c.encodeError(err, format, w, handler)
	}
}

func (c *ApiController) acceptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (accepted bool, format string, publish bool) {
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

func (c *ApiController) encodeResult(
	res     any,
	format  string,
	w       http.ResponseWriter,
	handler miruken.Handler,
) {
	w.Header().Set("Content-Type", format)
	out := io.Writer(w)
	msg := api.Message{Payload: res}
	if _, err := maps.MapInto(handler, msg, &out, maps.To(format)); err != nil {
		c.encodeError(err, format, w, handler)
	}
}

func (c *ApiController) encodeError(
	err     error,
	format  string,
	w       http.ResponseWriter,
	handler miruken.Handler,
) {
	w.Header().Set("Content-Type", format)
	statusCode := http.StatusInternalServerError
	handler     = miruken.BuildUp(handler, miruken.BestEffort)
	if sc, _, sce := maps.Map[int](handler, err, toStatusCode); sc != 0 && sce == nil {
		statusCode = sc
	}
	w.WriteHeader(statusCode)
	out := io.Writer(w)
	msg := api.Message{Payload: err}
	_, _ = maps.MapInto(handler, msg, &out, maps.To(format))
}

// Api creates a new ApiController to serve http routed messages.
func Api(ctx *context.Context) *ApiController {
	if ctx == nil {
		panic("ctx cannot be nil")
	}
	return &ApiController{ctx}
}

var toStatusCode = maps.To("http:status-code")
