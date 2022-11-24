package miruken

import (
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// Creates instances covariantly.
type Creates struct {
	CallbackBase
	key any
}

func (c *Creates) Key() any {
	return c.key
}

func (c *Creates) Policy() Policy {
	return _createsPolicy
}

func (c *Creates) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	count := c.ResultCount()
	return DispatchPolicy(handler, c, greedy, composer).
		OtherwiseHandledIf(c.ResultCount() > count)
}

// CreatesBuilder builds Creates callbacks.
type CreatesBuilder struct {
	CallbackBuilder
	key any
}

func (b *CreatesBuilder) WithKey(
	key any,
) *CreatesBuilder {
	if IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
}

func (b *CreatesBuilder) NewCreation() *Creates {
	return &Creates{
		CallbackBase: b.CallbackBase(),
		key: b.key,
	}
}

func Create[T any](
	handler         Handler,
	constraints ... any,
) (T, *promise.Promise[T], error) {
	return CreateKey[T](handler, TypeOf[T](), constraints...)
}

func CreateKey[T any](
	handler         Handler,
	key             any,
	constraints ... any,
) (t T, tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithKey(key).
			WithConstraints(constraints...)
	creates := builder.NewCreation()
	if result := handler.Handle(creates, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = &NotHandledError{creates}
	} else {
		_, tp, err = CoerceResult[T](creates, &t)
	}
	return
}

func CreateAll[T any](
	handler         Handler,
	constraints ... any,
) (t []T, tp *promise.Promise[[]T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithKey(TypeOf[T]()).
			WithConstraints(constraints...)
	creates := builder.NewCreation()
	if result := handler.Handle(creates, true, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		_, tp, err = CoerceResults[T](creates, &t)
	}
	return
}


// createsPolicy for creating instances covariantly.
type createsPolicy struct {
	CovariantPolicy
}

func (c *createsPolicy) NewConstructorBinding(
	handlerType reflect.Type,
	constructor *reflect.Method,
	spec        *policySpec,
) (binding Binding, err error) {
	return newConstructorBinding(handlerType, constructor, spec, spec != nil)
}

var _createsPolicy Policy = &createsPolicy{}