package miruken

import (
	"github.com/miruken-go/miruken/promise"
	"math"
	"reflect"
)

// initializer is a Filter that invokes a 'Constructor'
// method on the current output of the pipeline execution.
type initializer struct {
	constructor reflect.Method
	args        []arg
}

func (i *initializer) Order() int {
	return math.MaxInt32
}

func (i *initializer) Next(
	_        Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if out, pout, err = next.Pipe(); err != nil || len(out) == 0 {
		// no results so nothing to initialize
		return
	}
	if pout != nil {
		// wait for asynchronous results
		pout = promise.Then(pout, func(oo []any) []any {
			if len(oo) > 0 {
				return mergeOutputAwait(i.construct(ctx, oo[0]))
			}
			// no results so nothing to initialize
			return oo
 		})
	} else if _, pout, err = mergeOutput(i.construct(ctx, out[0])); err == nil && pout != nil {
		// asynchronous constructor so wait for completion
		pout = promise.Then(pout, func([]any) []any {
			return out
		})
	}
	return
}

func (i *initializer) construct(
	ctx  HandleContext,
	recv any,
) ([]any, *promise.Promise[[]any], error) {
	ctx.handler = recv
	return callFunc(i.constructor.Func, ctx, i.args, recv)
}

// initializerProvider is a FilterProvider for initializer.
type initializerProvider struct {
	filters []Filter
}

func (i *initializerProvider) Required() bool {
	return true
}

func (i *initializerProvider) AppliesTo(
	callback Callback,
) bool {
	switch callback.(type) {
	case *Provides, *Creates: return true
	default: return false
	}
}

func (i *initializerProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return i.filters, nil
}