package command

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/es/internal"
)

// Model captures a command specification.
type Model string

func (e *Model) Name() string {
	return string(*e)
}

func (e *Model) InitWithTag(tag reflect.StructTag) error {
	if command, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(command, "name=%s", e)
		return err
	}
	return nil
}

func (e *Model) InitWithBinding(binding miruken.Binding) error {
	if *e == "" {
		*e = Model(internal.DefaultBindingName(binding))
	}
	return nil
}