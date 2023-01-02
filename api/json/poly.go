package json

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
	"reflect"
)

type (
	// TypeFieldInfo defines the metadata for describing polymorphic messages.
	TypeFieldInfo struct {
		Field string
		Value string
	}

	// GoTypeFieldMapper provides TypeFieldInfo from  fully qualified package and name.
	GoTypeFieldMapper struct {}

	// message is the internal envelope for polymorphic json payloads.
	message struct {
		Payload *typeContainer `json:"payload"`
	}

	// messageMapper provides serialization/deserialization of message's.
	messageMapper struct {}
)

var (
	// ToTypeInfo requests type information for a type.
	ToTypeInfo = miruken.To("type:info")

	// KnownTypeFields holds the list of json property names
	// that can contain type discriminators.
	KnownTypeFields = []string{"$type", "@type"}
)

// GoTypeFieldMapper

func (m *GoTypeFieldMapper) GoTypeInfo(
	_*struct{
		miruken.Maps
		miruken.Format `to:"type:info"`
	  }, maps *miruken.Maps,
) (TypeFieldInfo, error) {
	typ := reflect.TypeOf(maps.Source())
	return TypeFieldInfo{"@type", typ.String()}, nil
}


// messageMapper

func (m *messageMapper) EncodeApiMessage(
	_*struct{
		miruken.Maps
		miruken.Format `to:"application/json"`
	  }, msg api.Message,
	maps *miruken.Maps,
	ctx  miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := maps.Target().(*io.Writer); ok {
		enc := json.NewEncoder(*writer)
		pay := typeContainer{msg.Payload, ctx.Composer()}
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
	ctx miruken.HandleContext,
) (api.Message, error) {
	var payload any
	pay := typeContainer{&payload,ctx.Composer()}
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