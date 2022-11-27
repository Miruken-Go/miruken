package http

import (
	"bytes"
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
)

const defaultContentType = "application/json"

func encodePayload(
	payload  any,
	format   string,
	writer   io.Writer,
	composer miruken.Handler,
) (io.Reader, error) {
	var buf bytes.Buffer
	stream := io.Writer(&buf)
	if _, err := miruken.MapInto(
		miruken.BuildUp(composer, _polyOptions),
		payload, &stream, miruken.To(format)); err != nil {
		return nil, err
	} else {
		w := writer
		var r io.Reader
		if writer == nil {
			var buf bytes.Buffer
			w, r = &buf, &buf
		}
		enc := json.NewEncoder(w)
		pay := buf.Bytes()
		msg := Message{(*json.RawMessage)(&pay)}
		if err := enc.Encode(msg); err != nil {
			return nil, err
		}
		return r, nil
	}
}

func decodePayload(
	payload  io.Reader,
	format   string,
	composer miruken.Handler,
) (any, error) {
	var msg Message
	decoder := json.NewDecoder(payload)
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	} else if pay := msg.Payload; pay == nil {
		return nil, nil
	} else {
		data, _, err := miruken.Map[any](
			miruken.BuildUp(composer, _polyOptions),
			bytes.NewReader(*pay), miruken.From(format))
		return data, err
	}
}

var (
	_polyOptions = miruken.Options(api.PolymorphicOptions{
		PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot),
	})
)