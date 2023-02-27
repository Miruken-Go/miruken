package httpsrv

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/provides"
	"io"
	"mime/multipart"
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
	accepted, contentType, from, boundary, publish := c.acceptRequest(w, r)
	if !accepted {
		return
	}
	to := from.FlipDirection()

	child := c.ctx.NewChild()
	defer child.Dispose()
	handler := miruken.BuildUp(child, api.Polymorphic)

	var payload any
	var body any = r.Body

	if len(boundary) > 0 {
		mr := multipart.NewReader(r.Body, boundary)
		if pc, _, err := maps.Map[api.PartContainer](handler, mr, from); err != nil {
			c.encodeError(err, true, contentType, to, w, handler)
			return
		} else if mainKey := pc.MainKey(); len(mainKey) > 0 {
			if main := pc.Part(mainKey); main != nil {
				contentType = main.ContentType()
				if len(contentType) == 0 {
					http.Error(w, fmt.Sprintf("400 missing %q part 'Content-Type' header", mainKey), http.StatusBadRequest)
					return
				}
				from    = maps.From(contentType, nil)
				to      = maps.To(contentType, nil)
				body    = main.Content()
				handler = miruken.BuildUp(handler, provides.With(pc))
			} else {
				http.Error(w, fmt.Sprintf("400 missing %q part", mainKey), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "400 unknown main part", http.StatusBadRequest)
			return
		}
	}

	if msg, _, err := maps.Map[api.Message](handler, body, from); err != nil {
		c.encodeError(err, true, contentType, to, w, handler)
		return
	} else if payload = msg.Payload; payload == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if publish {
		if pv, err := api.Publish(handler, payload); err != nil {
			c.encodeError(err, false, contentType, to, w, handler)
		} else if pv == nil {
			c.encodeResult(nil, contentType, to, w, handler)
		} else if _, err = pv.Await(); err == nil {
			c.encodeResult(nil, contentType, to, w, handler)
		} else {
			c.encodeError(err, false, contentType, to, w, handler)
		}
		return
	}

	if res, pr, err := api.Send[any](handler, payload); err != nil {
		c.encodeError(err, false, contentType, to, w, handler)
	} else if pr == nil {
		c.encodeResult(res, contentType, to, w, handler)
	} else if res, err = pr.Await(); err == nil {
		c.encodeResult(res, contentType, to, w, handler)
	} else {
		c.encodeError(err, false, contentType, to, w, handler)
	}
}

func (c *ApiController) acceptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (accepted bool, contentType string, format *maps.Format, boundary string, publish bool) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	contentType = r.Header.Get("Content-Type")
	if len(contentType) == 0 {
		http.Error(w, "400 missing 'Content-Type' header", http.StatusBadRequest)
		return
	}
	format, err := api.ParseContentType(contentType, maps.DirectionFrom)
	if err != nil {
		http.Error(w, "415 invalid 'Content-Type' header", http.StatusUnsupportedMediaType)
		return
	}
	if strings.HasPrefix(format.Name(), "multipart/") {
		if b, ok := format.Params()["boundary"]; !ok {
			http.Error(w, http.ErrMissingBoundary.Error(), http.StatusUnsupportedMediaType)
			return
		} else {
			boundary = b
		}
	}
	path := r.RequestURI
	if path == "/process" || strings.HasPrefix(path, "/process/") {
		accepted = true
		publish  = false
	} else if path == "/publish" || strings.HasPrefix(path, "/publish/") {
		accepted = true
		publish  = false
	} else {
		http.Error(w, "404 not found", http.StatusNotFound)
	}
	return
}

func (c *ApiController) encodeResult(
	res         any,
	contentType string,
	format      *maps.Format,
	w           http.ResponseWriter,
	handler     miruken.Handler,
) {
	w.Header().Set("Content-Type", contentType)
	out := io.Writer(w)
	msg := api.Message{Payload: res}
	if _, err := maps.MapInto(handler, msg, &out, format); err != nil {
		c.encodeError(err, false, contentType, format, w, handler)
	}
}

func (c *ApiController) encodeError(
	err         error,
	mapping     bool,
	contentType string,
	format      *maps.Format,
	w           http.ResponseWriter,
	handler     miruken.Handler,
) {
	if mapping {
		var nh *miruken.NotHandledError
		if errors.As(err, &nh) {
			http.Error(w, "415 invalid 'Content-Type' header", http.StatusUnsupportedMediaType)
			return
		}
	}
	w.Header().Set("Content-Type", contentType)
	statusCode := http.StatusInternalServerError
	handler = miruken.BuildUp(handler, miruken.BestEffort)
	if sc, _, sce := maps.Map[int](handler, err, toStatusCode); sc != 0 && sce == nil {
		statusCode = sc
	}
	w.WriteHeader(statusCode)
	out := io.Writer(w)
	msg := api.Message{Payload: err}
	_, _ = maps.MapInto(handler, msg, &out, format)
}

// Api creates a new ApiController to serve http routed messages.
func Api(ctx *context.Context) *ApiController {
	if ctx == nil {
		panic("ctx cannot be nil")
	}
	return &ApiController{ctx}
}

var toStatusCode = maps.To("http:status-code", nil)
