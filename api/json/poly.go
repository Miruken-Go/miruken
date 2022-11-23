package json

import (
	"github.com/miruken-go/miruken"
	"reflect"
)

type (
	TypeFieldHandling uint8

	TypeFieldInfo struct {
		Field string
		Value string
	}

	GoTypeFieldMapper struct {}

	PolymorphicOptions struct {
		KnownTypeFields []string
	}
)

var defaultTypeFields = []string{"$type", "@type"}

const (
	TypeFieldHandlingNone TypeFieldHandling = 0
	TypeFieldHandlingRoot TypeFieldHandling = 1 << iota
)

// GoTypeFieldMapper

func (m *GoTypeFieldMapper) DefaultTypeInfo(
	maps *miruken.Maps,
) (TypeFieldInfo, miruken.HandleResult) {
	typ := reflect.TypeOf(maps.Source())
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if name := typ.Name(); len(name) == 0 {
		return TypeFieldInfo{}, miruken.NotHandled
	} else {
		return TypeFieldInfo{"@type", typ.String()}, miruken.Handled
	}
}
