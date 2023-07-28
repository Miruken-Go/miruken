package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
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
	return createsPolicyInstance
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

func (c *Creates) String() string {
	return fmt.Sprintf("creates => %+v", c.key)
}

// CreatesBuilder builds Creates callbacks.
type CreatesBuilder struct {
	CallbackBuilder
	key any
}

func (b *CreatesBuilder) WithKey(
	key any,
) *CreatesBuilder {
	if internal.IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
}

func (b *CreatesBuilder) New() *Creates {
	return &Creates{
		CallbackBase: b.CallbackBase(),
		key: b.key,
	}
}

func Create[T any](
	handler     Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return CreateKey[T](handler, internal.TypeOf[T](), constraints...)
}

func CreateKey[T any](
	handler     Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithKey(key).
		    IntoTarget(&t).
			WithConstraints(constraints...)
	creates := builder.New()
	if result := handler.Handle(creates, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = &NotHandledError{creates}
	} else if _, p := creates.Result(false); p != nil {
		tp = promise.Coerce[T](p)
	}
	return
}

func CreateAll[T any](
	handler     Handler,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithKey(internal.TypeOf[T]()).
		    IntoTarget(&t).
			WithConstraints(constraints...)
	creates := builder.New()
	if result := handler.Handle(creates, true, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		if _, p := creates.Result(true); p != nil {
			tp = promise.Then(p, func(any) []T {
				return t
			})
		}
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
	spec        *bindingSpec,
	key         any,
) (binding Binding, err error) {
	return newConstructorBinding(handlerType, constructor, spec, key, spec != nil)
}

var createsPolicyInstance Policy = &createsPolicy{}