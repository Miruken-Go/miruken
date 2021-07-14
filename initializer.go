package miruken

import "math"

// initializer is a Filter that invokes an 'initialize' method
// on the current result of the pipeline.
type initializer struct {
	initMethod methodInvoke
}

func (i *initializer) Order() int {
	return math.MaxInt32
}

func (i *initializer) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	instance, err := next.Filter()
	if err == nil && len(instance) > 0 {
		_, err = i.initMethod.Invoke(instance[0], context)
	}
	return instance, err
}

// initializerProvider is a FilterProvider for initializer.
type initializerProvider struct {
	filters []Filter
}

func (i *initializerProvider) Required() bool {
	return true
}

func (i *initializerProvider) AppliesTo(
	callback interface{},
) bool {
	switch callback.(type) {
	case *Inquiry, *Creates: return true
	default: return false
	}
}

func (i *initializerProvider) Filters(
	binding  Binding,
	callback interface{},
	composer Handler,
) ([]Filter, error) {
	return i.filters, nil
}