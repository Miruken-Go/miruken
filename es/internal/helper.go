package internal

import (
	"reflect"

	"github.com/miruken-go/miruken"
)

// DefaultBindingName returns the default name for a binding
// based on the binding key.
// If the key is a `reflect.Type`, the type name is returned
// with the pointer prefix removed.
// Otherwise, an empty string is returned.
func DefaultBindingName(binding miruken.Binding) string {
	if typ, ok := binding.Key().(reflect.Type); ok {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		return typ.String()
	}
	return ""
}
