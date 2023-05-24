package httpsrv

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/slices"
	"github.com/timewasted/go-accept-headers"
	"io"
	"net/http"
	"net/textproto"
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
		h.logger = logr.Discard()
	} else {
		h.logger = logger
	}
	h.ctx = ctx
}

func (h *ApiHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer h.handlePanic(w)

	accepted, from, publish := h.acceptRequest(w, r)
	if !accepted {
		return
	}

	child := h.ctx.NewChild()
	defer child.Dispose()
	handler := miruken.BuildUp(child, api.Polymorphic)

	msg, _, _, err := maps.Out[api.Message](handler, r.Body, from)
	if err != nil {
		h.encodeError(err, http.StatusUnsupportedMediaType, w, handler)
		return
	}

	payload := msg.Payload
	if payload == nil {
		http.Error(w, "400 missing payload", http.StatusBadRequest)
		return
	}

	if c, ok := payload.(api.Content); ok {
		if payload = c.Body(); miruken.IsNil(payload) {
			http.Error(w, "400 missing content body", http.StatusBadRequest)
			return
		}
		handler = miruken.BuildUp(handler, provides.With(c))
	}

	if publish {
		if pv, err := api.Publish(handler, payload); err != nil {
			h.encodeError(err, 0, w, handler)
		} else if pv == nil {
			h.encodeResult(nil, r, w, handler)
		} else if _, err = pv.Await(); err == nil {
			h.encodeResult(nil, r, w, handler)
		} else {
			h.encodeError(err, 0, w, handler)
		}
	} else {
		if res, pr, err := api.Send[any](handler, payload); err != nil {
			h.encodeError(err, 0, w, handler)
		} else if pr == nil {
			h.encodeResult(res, r, w, handler)
		} else if res, err = pr.Await(); err == nil {
			h.encodeResult(res, r, w, handler)
		} else {
			h.encodeError(err, 0, w, handler)
		}
	}
}

func (h *ApiHandler) acceptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (accepted bool, format *maps.Format, publish bool) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	contentType := r.Header.Get("Content-Type")
	if len(contentType) == 0 {
		http.Error(w, "400 missing 'Content-Type' header", http.StatusBadRequest)
		return
	}
	format, err := api.ParseMediaType(contentType, maps.DirectionFrom)
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
	result  any,
	r       *http.Request,
	w       http.ResponseWriter,
	handler miruken.Handler,
) {
	header := w.Header()
	var formats []*maps.Format
	if content, ok := result.(api.Content); ok {
		if format, err := api.ParseMediaType(content.MediaType(), maps.DirectionTo); err == nil {
			formats = []*maps.Format{format}
			if wb := content.(interface{ WriteBody() any }); ok {
				result = wb.WriteBody()
			} else {
				result = content.Body()
			}
		} else {
			h.encodeError(err, 0, w, handler)
			return
		}
		api.MergeHeader(textproto.MIMEHeader(header), content.Metadata())
	} else if hdr := r.Header.Get("Accept"); hdr != "" {
		if fs := accept.Parse(hdr); len(fs) > 0 {
			formats = slices.Map[accept.Accept, *maps.Format](fs, formatAccept)
		}
	}
	if len(formats) == 0 {
		formats = []*maps.Format{api.ToJson}
	}
	msg := api.Message{Payload: result}
	if len(formats) == 1 && formats[0].Rule() == maps.FormatRuleEquals {
		format := formats[0]
		header.Set("Content-Type", format.Name())
		out := io.Writer(w)
		if _, _, err := maps.Into(handler, msg, &out, format); err != nil {
			h.encodeError(err, http.StatusNotAcceptable, w, handler)
		}
	} else {
		for i, format := range formats {
			var b bytes.Buffer
			out := io.Writer(&b)
			if _, m, err := maps.Into(handler, msg, &out, format); err == nil {
				var contentType string
				if format.Rule() == maps.FormatRuleEquals {
					contentType = api.FormatMediaType(format)
				} else if match := m.Matched(); match != nil {
					contentType = api.FormatMediaType(match)
				} else {
					w.WriteHeader(http.StatusNotAcceptable)
					return
				}
				header.Set("Content-Type", contentType)
				if _, err := w.Write(b.Bytes()); err != nil {
					h.logger.Error(err, "unable to write response")
					w.WriteHeader(http.StatusInternalServerError)
				}
				break
			} else if i == len(formats)-1 {
				h.encodeError(err, http.StatusNotAcceptable, w, handler)
			}
		}
	}
}

func (h *ApiHandler) encodeError(
	err                  error,
	notHandledStatusCode int,
	w                    http.ResponseWriter,
	handler              miruken.Handler,
) {
	if notHandledStatusCode > 0 {
		var nh *miruken.NotHandledError
		if errors.As(err, &nh) {
			w.WriteHeader(notHandledStatusCode)
			return
		}
	}
	w.Header().Set("Content-Type", api.ToJson.Name())
	statusCode := http.StatusInternalServerError
	handler = miruken.BuildUp(handler, miruken.BestEffort)
	if sc, _, _, e := maps.Out[int](handler, err, toStatusCode); sc != 0 && e == nil {
		statusCode = sc
	}
	w.WriteHeader(statusCode)
	out := io.Writer(w)
	msg := api.Message{Payload: err}
	_, _, _ = maps.Into(handler, msg, &out, api.ToJson)
}

func formatAccept(a accept.Accept) *maps.Format {
	var sb strings.Builder
	if a.Subtype == "*" {
		if a.Type == "*" {
			return maps.To("*", a.Extensions)
		}
		sb.WriteString("/")
	}
	sb.WriteString(a.Type)
	if a.Subtype != "*" {
		sb.WriteString("/")
		sb.WriteString(a.Subtype)
	} else {
		sb.WriteString("//")
	}
	return maps.To(sb.String(), a.Extensions)
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
