package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"net/http"
)

type (
	Controller struct {
		miruken.ContextualBase
	}
)

func (c *Controller) Constructor(
	_*struct{
		miruken.Provides
		miruken.Scoped
	  },
) {
}

func (c *Controller) SetContext(ctx *miruken.Context) {
	c.ChangeContext(c, ctx)
}

func (c *Controller) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := c.Context()
	if ctx == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ctx = ctx.NewChild()
	defer ctx.Dispose()
	format := r.Header.Get("Content-Type")
	if len(format) == 0 {
		format = defaultContentType
	}
	if payload, err := decodePayload(r.Body, format, ctx); err != nil {
		c.encodeError(err, r, w)
	} else if payload == nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if res, pr, err := api.Send[any](ctx, payload); err != nil {
			c.encodeError(err, r, w)
		} else if pr == nil {
			c.encodeResult(res, format, r, w)
		} else if res, err = pr.Await(); err == nil {
			c.encodeResult(res, format, r, w)
		} else {
			c.encodeError(err, r, w)
		}
	}
}

func (c *Controller) encodeResult(
	res      any,
	format   string,
	r        *http.Request,
	w        http.ResponseWriter,
) {
	w.Header().Set("Content-Type", format)
	if _, err := encodePayload(res, format, w, c.Context()); err != nil {
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