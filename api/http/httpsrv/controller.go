package httpsrv

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/provides"
	"io"
	"net/http"
	"runtime"
	"strings"
)

// ApiHandler is an http.Handler for processing api requests.
type ApiHandler struct {
	ctx    *context.Context
	logger logr.Logger
}

func (h *ApiHandler) Constructor(
	_*struct{provides.It; context.Lifestyle},
	_*struct{args.Optional}, logger logr.Logger,
	ctx *context.Context,
) {
	if logger == h.logger {
		logger = logr.Discard()
	}
	h.logger = logger
	h.ctx    = ctx
}

func (h *ApiHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer h.handlePanic(w)

	accepted, contentType, from, publish := h.acceptRequest(w, r)
	if !accepted {
		return
	}
	to := from.FlipDirection()

	child := h.ctx.NewChild()
	defer child.Dispose()
	handler := miruken.BuildUp(child, api.Polymorphic)

	msg, _, err := maps.Out[api.Message](handler, r.Body, from)
	if err != nil {
		h.encodeError(err, true, contentType, to, w, handler)
		return
	}

	payload := msg.Payload
	if payload == nil {
		http.Error(w, "400 missing payload", http.StatusBadRequest)
		return
	}

	if pc, ok := payload.(api.PartContainer); ok {
		if main := pc.MainPart(); main != nil {
			if payload = main.Content(); miruken.IsNil(payload) {
				http.Error(w, "400 empty main part", http.StatusBadRequest)
				return
			}
			contentType = main.ContentType()
			to          = maps.To(contentType, nil)
			handler     = miruken.BuildUp(handler, provides.With(pc))
		} else {
			http.Error(w, "400 missing main part", http.StatusBadRequest)
			return
		}
	}

	if publish {
		if pv, err := api.Publish(handler, payload); err != nil {
			h.encodeError(err, false, contentType, to, w, handler)
		} else if pv == nil {
			h.encodeResult(nil, contentType, to, w, handler)
		} else if _, err = pv.Await(); err == nil {
			h.encodeResult(nil, contentType, to, w, handler)
		} else {
			h.encodeError(err, false, contentType, to, w, handler)
		}
	} else {
		if res, pr, err := api.Send[any](handler, payload); err != nil {
			h.encodeError(err, false, contentType, to, w, handler)
		} else if pr == nil {
			h.encodeResult(res, contentType, to, w, handler)
		} else if res, err = pr.Await(); err == nil {
			h.encodeResult(res, contentType, to, w, handler)
		} else {
			h.encodeError(err, false, contentType, to, w, handler)
		}
	}
}

func (h *ApiHandler) acceptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (accepted bool, contentType string, format *maps.Format, publish bool) {
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

func (h *ApiHandler) encodeResult(
	res         any,
	contentType string,
	format      *maps.Format,
	w           http.ResponseWriter,
	handler     miruken.Handler,
) {
	w.Header().Set("Content-Type", contentType)
	out := io.Writer(w)
	msg := api.Message{Payload: res}
	if _, err := maps.Into(handler, msg, &out, format); err != nil {
		h.encodeError(err, false, contentType, format, w, handler)
	}
}

func (h *ApiHandler) encodeError(
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
	if sc, _, sce := maps.Out[int](handler, err, toStatusCode); sc != 0 && sce == nil {
		statusCode = sc
	}
	w.WriteHeader(statusCode)
	out := io.Writer(w)
	msg := api.Message{Payload: err}
	_, _ = maps.Into(handler, msg, &out, format)
}

func (h *ApiHandler) handlePanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		err, _ := r.(error)
		buf := make([]byte, 2048)
		n := runtime.Stack(buf, false)
		buf = buf[:n]
		msg := fmt.Sprintf("%v", r)
		h.logger.Error(err, "recovering from http panic", "stack", string(buf))
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

// Handler returns a http.Handler for processing api calls
// bound to the given miruken.Handler.
func Handler(handler miruken.Handler) http.Handler {
	if _, ok := handler.(*context.Context); !ok {
		handler = context.New(handler)
	}
	h, cp, err := provides.Type[*ApiHandler](handler)
	if err != nil {
		panic(err)
	}
	if cp != nil {
		if h, err = cp.Await(); err != nil {
			panic(err)
		}
	}
	return h
}

var toStatusCode = maps.To("http:status-code", nil)
