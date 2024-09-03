package internal

import (
	"reflect"

	"github.com/google/uuid"
)

// DefaultTypeName returns the default name for a type.
func DefaultTypeName(typ reflect.Type) string {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.String()
}

var (
	UUIDType = reflect.TypeFor[uuid.UUID]()
)
