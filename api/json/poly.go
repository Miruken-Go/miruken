package json

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
)

type (
	// message is the internal envelope for polymorphic json payloads.
	message struct {
		Payload *typeContainer `json:"payload"`
	}

	// messageMapper provides serialization/deserialization of message's.
	messageMapper struct {}
)

var (
	// KnownTypeFields holds the list of json property names
	// that can contain type discriminators.
	KnownTypeFields = []string{"$type", "@type"}
)


// messageMapper

func (m *messageMapper) EncodeApiMessage(
	_*struct{
		miruken.Maps
		miruken.Format `to:"application/json"`
	  }, msg api.Message,
	maps *miruken.Maps,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions api.PolymorphicOptions,
	ctx  miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := maps.Target().(*io.Writer); ok {
		enc := json.NewEncoder(*writer)
		pay := typeContainer{
			v:        msg.Payload,
			typInfo:  polyOptions.TypeInfoFormat,
			composer: ctx.Composer(),
		}
		env := message{&pay}
		if err := enc.Encode(env); err == nil {
			return *writer, err
		}
	}
	return nil, nil
}

func (m *messageMapper) DecodeApiMessage(
	_*struct{
		miruken.Maps
		miruken.Format `from:"application/json"`
	  }, stream io.Reader,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options StdOptions,
	ctx miruken.HandleContext,
) (api.Message, error) {
	var payload any
	pay := typeContainer{
		v:        &payload,
		trans:    options.Transformers,
		composer: ctx.Composer(),
	}
	msg := message{&pay}
	dec := json.NewDecoder(stream)
	err := dec.Decode(&msg)
	return api.Message{Payload: payload}, err
}


var (
	// Polymorphic returns a miruken.Builder that enables polymorphic messaging.
	Polymorphic miruken.Builder = miruken.Options(api.PolymorphicOptions{
		PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot),
	})
)