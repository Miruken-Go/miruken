package json

import (
	"encoding/json"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"io"
	"reflect"
)

type (
	// TypeFieldInfo defines the metadata for encoding
	// polymorphic json message discriminators.
	TypeFieldInfo struct {
		Field string
		Value string
	}

	// GoTypeFieldMapper provides TypeFieldInfo using the
	// fully qualified package and type name.
	GoTypeFieldMapper struct {}

	// message is a json specific envelope for polymorphic payloads.
	message struct {
		Payload *typeContainer `json:"payload"`
	}

	// messageMapper provides the serialization/deserialization
	// of json polymorphic payloads.
	messageMapper struct {}
)

var (
	// ToTypeInfo request type information for a type.
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
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if name := typ.Name(); len(name) == 0 {
		return TypeFieldInfo{}, fmt.Errorf("no type info for anonymous %+v", typ)
	} else {
		return TypeFieldInfo{"@type", typ.String()}, nil
	}
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
