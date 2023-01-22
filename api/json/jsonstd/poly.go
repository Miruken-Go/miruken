package jsonstd

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
)

type (
	// apiMessage is the internal envelope for api messages.
	apiMessage struct {
		Payload *typeContainer `json:"payload"`
	}

	// apiMessageMapper maps api.Message to/from internal apiMessage.
	apiMessageMapper struct {}
)


// apiMessageMapper

func (m *apiMessageMapper) Encode(
	_*struct{
		miruken.Maps
		miruken.Format `to:"application/json"`
	  }, msg api.Message,
	maps *miruken.Maps,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	ctx miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := maps.Target().(*io.Writer); ok {
		enc := json.NewEncoder(*writer)
		pay := typeContainer{
			v:        msg.Payload,
			typInfo:  apiOptions.TypeInfoFormat,
			composer: ctx.Composer(),
		}
		env := apiMessage{&pay}
		if err := enc.Encode(env); err == nil {
			return *writer, err
		}
	}
	return nil, nil
}

func (m *apiMessageMapper) Decode(
	_*struct{
		miruken.Maps
		miruken.Format `from:"application/json"`
	  }, stream io.Reader,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	ctx miruken.HandleContext,
) (api.Message, error) {
	var payload any
	pay := typeContainer{
		v:        &payload,
		trans:    options.Transformers,
		composer: ctx.Composer(),
	}
	msg := apiMessage{&pay}
	dec := json.NewDecoder(stream)
	err := dec.Decode(&msg)
	return api.Message{Payload: payload}, err
}
