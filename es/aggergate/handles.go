package aggergate

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
)

type (
	// Handles marks a handler for event-sourcing rules.
	Handles struct {
		miruken.BindingGroup
		handles.It
		processor
	}

	// processor interprets the current input as a command acting
	// on an aggregate.
	// The output is interpreted as a sequence of events applied to
	// an aggregate.
	// Expects the argument after the command to be the aggregate.
	processor struct{
		name string
	}
)


// CommandModel

func (p *processor) Name() string {
	return p.name
}

func (p *processor) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &p.name)
		return err
	}
	return nil
}
