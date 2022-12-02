package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
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
	if payload, err := decodePayload(r.Body, format, ctx); err != nil {
		c.encodeError(err, r, w)
	} else if payload == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if publish {
		if pv, err := api.Publish(ctx, payload); err != nil {
			c.encodeError(err, r, w)
		} else if pv == nil {
			c.encodeResult(nil, format, r, w)
		} else if _, err = pv.Await(); err == nil {
			c.encodeResult(nil, format, r, w)
		} else {
			c.encodeError(err, r, w)
		}
	} else if res, pr, err := api.Send[any](ctx, payload); err != nil {
		c.encodeError(err, r, w)
	} else if pr == nil {
		c.encodeResult(res, format, r, w)
	} else if res, err = pr.Await(); err == nil {
		c.encodeResult(res, format, r, w)
	} else {
		c.encodeError(err, r, w)
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
	r        *http.Request,
	w        http.ResponseWriter,
) {
	w.Header().Set("Content-Type", format)
	if _, err := encodePayload(res, format, w, c.Context); err != nil {
		c.encodeError(err, r, w)
	}
}

func (c *Controller) encodeError(
	err error,
	r   *http.Request,
	w   http.ResponseWriter,
) {
	w.WriteHeader(http.StatusInternalServerError)
}