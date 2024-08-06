package aggergate

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Root marks a type as an aggregate root.
	// Aggregate roots are scoped to a context and provide metadata.
	Root struct {
		miruken.BindingGroup
		provides.It
		context.Scoped
		loader
	}

	// loader is responsible for retrieving the current state of an aggregate.
	loader struct{
		name string
	}
)


// Model

func (l *loader) Name() string {
	return l.name
}

func (l *loader) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("agg"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &l.name)
		return err
	}
	return nil
}
