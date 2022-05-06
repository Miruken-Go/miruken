package miruken

import (
	"math"
	"reflect"
)

// initializer is a Filter that invokes a 'Constructor'
// method on the current result of the pipeline.
type initializer struct {
	constructor reflect.Method
	args        []arg
}

func (i *initializer) Order() int {
	return math.MaxInt32
}

func (i *initializer) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]any, error) {
	instance, err := next.Filter()
	if err == nil && len(instance) > 0 {
		var results []any
		results, err = i.invoke(context, instance[0])
		if len(results) > 0 {
			if e, ok := results[len(results)-1].(error); ok {
				err = e
			}
		}
	}
	return instance, err
}

func (i *initializer) invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	ctor := i.constructor
	if results, err := callFunc(ctor.Func, context, i.args, explicitArgs...); err != nil {
		return nil, MethodBindingError{ctor, err}
	} else {
		return results, nil
	}
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