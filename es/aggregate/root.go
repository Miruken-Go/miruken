package aggregate

import (
	"fmt"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Root marks a type as an aggregate root.
	// Aggregate roots are scoped to a context and provide metadata.
	Root struct {
		miruken.BindingGroup
		provides.It
		Export
		loadProvider
	}

	// loader is a miruken.Filter for retrieving the current state of an aggregate.
	loader struct{}

	// loadProvider is a miruken.FilterProvider for loader.
	loadProvider struct {}
)


// loader

func (l loader) Order() int {
	return miruken.FilterStageCreation-1000
}

func (l loader) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	if _, ok := provider.(*loadProvider); ok {
		// Receiver is always created synchronously
		if out, _, err = next.Pipe(); err == nil && len(out) > 0 {
			options, ok := miruken.GetOptions[Options](ctx)
			if ok {
				fmt.Println(options)
			}
			return
		}
	}
	return next.Abort()
}


// loadProvider

func (l *loadProvider) Required() bool {
	return true
}

func (l *loadProvider) AppliesTo(
	callback miruken.Callback,
) bool {
	switch callback.(type) {
	case *miruken.Provides, *miruken.Creates:
		return true
	default:
		return false
	}
}

func (l *loadProvider) Filters(
	miruken.Binding, any,
	miruken.Handler,
) ([]miruken.Filter, error) {
	return loadFilters, nil
}


var loadFilters = []miruken.Filter{loader{}}