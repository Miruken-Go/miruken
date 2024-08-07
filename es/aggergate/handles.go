package aggergate

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Handles marks a handler for event-sourcing rules.
	Handles struct {
		miruken.BindingGroup
		handles.It
		processProvider
	}

	// processor interprets the current input as a command acting
	// on an aggregate.
	// The output is interpreted as a sequence of events applied to
	// an aggregate.
	// Expects the argument after the command to be the aggregate.
	processor struct{}

	// processProvider is a miruken.FilterProvider for processor.
	processProvider struct {
		name string
	}
)


// processor

func (p processor) Order() int {
	return 10
}

func (p processor) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	if out, pout, err = next.Pipe(); err == nil && len(out) > 0 {
	}
	return
}


// processProvider

func (p *processProvider) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &p.name)
		return err
	}
	return nil
}

func (p *processProvider) Required() bool {
	return true
}

func (p *processProvider) AppliesTo(
	callback miruken.Callback,
) bool {
	_, ok := callback.(*handles.It)
	return ok
}

func (p *processProvider) Filters(
	miruken.Binding, any,
	miruken.Handler,
) ([]miruken.Filter, error) {
	return processFilters, nil
}


var processFilters = []miruken.Filter{processor{}}
