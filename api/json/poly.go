package json

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"reflect"
)

type (
	TypeFieldInfo struct {
		Field string
		Value string
	}

	GoTypeFieldMapper struct {}
)

var KnownTypeFields = []string{"$type", "@type"}

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
