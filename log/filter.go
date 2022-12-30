package log

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// Provider is a FilterProvider for logging.
	Provider struct {
		verbosity int
	}

	// filter logs basic callback execution details.
	filter struct {}
)


// Provider

func (l *Provider) InitWithTag(tag reflect.StructTag) error {
	if log, ok := tag.Lookup("log"); ok {
		_, err := fmt.Sscanf(log, "verbosity=%d", &l.verbosity)
		return err
	}
	return nil
}

func (l *Provider) Required() bool {
	return false
}

func (l *Provider) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return _filters, nil
}

// NewProvider builds a new Provider for logging.
// verbosity is used to control the level of logging.
func NewProvider(verbosity int) *Provider {
	return &Provider{verbosity}
}


// filter

func (f filter) Order() int {
	return miruken.FilterStageLogging
}

func (f filter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if _, ok := provider.(*Provider); ok {
		// perform the next step in the pipeline
		if out, pout, err = next.Pipe(); err != nil {
			return
		} else if pout == nil {
			return
		} else {
			// asynchronous output validation
			return nil, promise.Then(pout, func(oo []any) []any {
				if len(oo) > 0 && !miruken.IsNil(oo[0]) {
				}
				return oo
			}), nil
		}
	}
	return next.Abort()
}

var _filters = []miruken.Filter{filter{}}
