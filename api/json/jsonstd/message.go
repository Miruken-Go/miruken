package jsonstd

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
	"io"
)

// MessageSurrogate is a json standard surrogate for api.Message.
type MessageSurrogate struct {
	Payload json.RawMessage `json:"payload"`
}


func (m *SurrogateMapper) EncodeMessage(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, msg api.Message,
	it *maps.It,
	ctx miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := it.Target().(*io.Writer); ok {
		var sur MessageSurrogate
		if payload := msg.Payload; payload != nil {
			pj, _, err := maps.Map[string](ctx.Composer(), msg.Payload, api.ToJson)
			if err != nil {
				return nil, err
			}
			sur.Payload = json.RawMessage(pj)
		}
		enc := json.NewEncoder(*writer)
		if err := enc.Encode(sur); err == nil {
			return *writer, err
		}
	}
	return nil, nil
}

func (m *SurrogateMapper) DecodeMessage(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, stream io.Reader,
	ctx miruken.HandleContext,
) (api.Message, error) {
	var sur MessageSurrogate
	dec := json.NewDecoder(stream)
	if err := dec.Decode(&sur); err != nil {
		return api.Message{}, err
	}
	composer := ctx.Composer()
	payload, _, err := maps.Map[any](composer, string(sur.Payload), api.FromJson)
	if sur, ok := payload.(api.Surrogate); ok {
		payload, err = sur.Original(composer)
	}
	return api.Message{Payload: payload}, err
}