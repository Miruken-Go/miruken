package aggregate

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
)

// Model captures aggregate details.
type Model string

func (e *Model) Name() string {
	return string(*e)
}

func (e *Model) InitWithTag(tag reflect.StructTag) error {
	if aggregate, ok := tag.Lookup("aggregate"); ok {
		_, err := fmt.Sscanf(aggregate, "name=%s", e)
		return err
	}
	return nil
}

func (e *Model) InitWithBinding(binding miruken.Binding) error {
	if *e == "" {
		if typ, ok := binding.Key().(reflect.Type); ok {
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			*e = Model(typ.String())
		}
	}
	return nil
}