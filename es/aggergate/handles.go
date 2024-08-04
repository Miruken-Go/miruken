package aggergate

import (
	"fmt"
	"reflect"
)

type (
	// Handles is a FilterProvider that applies event-sourcing rules.
	Handles struct {
		name string
	}

	// filter interprets the current input as a command acting on an
	// aggregate and the output as a sequence of es to be applied
	// to the aggregate.
	// Expects the argument after the command to be the aggregate.
	filter struct{}
)


// Handles

func (h *Handles) Name() string {
	return h.name
}

func (h *Handles) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("handler"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &h.name)
		return err
	}
	return nil
}
