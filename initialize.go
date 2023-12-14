package miruken

import (
	"math"

	"github.com/miruken-go/miruken/promise"
)

type (
	// Init marks a method as an initializer.
	Init struct{}

	// initializer is a Filter that invokes a 'Constructor'
	// method and optional 'Init' methods on the current output
	// of the pipeline execution.
	// If a 'Constructor' is present, it will be the first item.
	// The remaining initializers are called in lexicographic order.
	initializer struct {
		inits []funcCall
	}

	// initProvider is a FilterProvider for initializer.
	initProvider struct {
		filters []Filter
	}
)

// initializer

func (i *initializer) Order() int {
	return math.MaxInt32
}

func (i *initializer) Next(
	self Filter,
	next Next,
	ctx HandleContext,
	provider FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	// Receiver is always created synchronously
	if out, _, err = next.Pipe(); err == nil && len(out) > 0 {
		pout, err = i.construct(ctx, out[0])
		if err == nil && pout != nil {
			// asynchronous constructor so wait for completion
			pout = promise.Then(pout, func([]any) []any {
				return out
			})
		}
	}
	return
}

func (i *initializer) construct(
	ctx HandleContext,
	recv any,
) (*promise.Promise[[]any], error) {
	ctx.Handler = recv
	for idx, init := range i.inits {
		_, pout, err := mergeOutput(init.Invoke(ctx, recv))
		if err != nil {
			return nil, err
		} else if pout != nil {
			return promise.Then(pout, func([]any) []any {
				// After the first promise, invoke remaining initializers inline.
				for _, next := range i.inits[idx+1:] {
					mergeOutputAwait(next.Invoke(ctx, recv))
				}
				return nil
			}), nil
		}
	}
	return nil, nil
}

// initProvider

func (i *initProvider) Required() bool {
	return true
}

func (i *initProvider) AppliesTo(
	callback Callback,
) bool {
	switch callback.(type) {
	case *Provides, *Creates:
		return true
	default:
		return false
	}
}

func (i *initProvider) Filters(
	binding Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return i.filters, nil
}
