package aggergate

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Root defines metadata for an aggregate root.
	Root struct {
		miruken.BindingGroup
		provides.It
		context.Scoped
		Metadata
	}

	// Metadata provides metadata for an aggregate root.
	Metadata struct {
		name string
	}
)


// Root

func (m *Metadata) Name() string {
	return m.name
}

func (m *Metadata) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("agg"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &m.name)
		return err
	}
	return nil
}
