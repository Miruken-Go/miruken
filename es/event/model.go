package event

import (
	"reflect"

	"github.com/miruken-go/miruken/es/internal"
)

// Model captures an event specification.
type Model struct {
	name string
	typ  reflect.Type
}


func (m *Model) Name() string {
	return m.name
}

func (m *Model) Type() reflect.Type {
	return m.typ
}


// NewModel creates an event model for type and name.
func NewModel(typ reflect.Type, name string) (*Model, error) {
	if typ == nil {
		panic("event: typ cannot be nil")
	}

	if name == "" {
		name = internal.DefaultTypeName(typ)
	}

	return &Model{
		name: name,
		typ:  typ,
	}, nil
}