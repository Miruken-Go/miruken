package miruken

import (
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// Creates instances covariantly.
type Creates struct {
	CallbackBase
	typ reflect.Type
}

func (c *Creates) Key() any {
	return c.typ
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
	typ reflect.Type
}

func (b *CreatesBuilder) WithType(
	typ reflect.Type,
) *CreatesBuilder {
	if IsNil(typ) {
		panic("type cannot be nil")
	}
	b.typ = typ
	return b
}

func (b *CreatesBuilder) NewCreation() *Creates {
	return &Creates{
		CallbackBase: b.CallbackBase(),
		typ: b.typ,
	}
}

func Create[T any](
	handler Handler,
) (t T, tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	creates := builder.
		WithType(TypeOf[T]()).
		NewCreation()
	if result := handler.Handle(creates, false, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		_, tp, err = CoerceResult[T](creates, &t)
	}
	return
}

func CreateAll[T any](
	handler Handler,
) (t []T, tp *promise.Promise[[]T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithType(TypeOf[T]())
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