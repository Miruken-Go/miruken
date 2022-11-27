package http

import (
	"bytes"
	"encoding/json"
	"github.com/miruken-go/miruken"
	"net/http"
)

type (
	Controller struct {
		miruken.ContextualBase
	}
)

func (c *Controller) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if payload, err := c.decodePayload(r); err != nil {
		c.encodeError(err, r, w)
	} else if payload == nil {

	} else {

	}
}

func (c *Controller) decodePayload(
	req *http.Request,
) (any, error) {
	var msg Message
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	} else if pay := msg.Payload; pay == nil {
		return nil, nil
	} else {
		contentType := req.Header.Get("Content-type")
		if len(contentType) == 0 {
			contentType = _defaultContentType
		}
		data, _, err := miruken.Map[any](
			miruken.BuildUp(c.Context(), _polyOptions),
			bytes.NewReader(*pay), &miruken.Format{As: contentType})
		return data, err
	}
}

func (c *Controller) encodeError(
	err      error,
	req      *http.Request,
	w        http.ResponseWriter,
) {

}