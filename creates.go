package miruken

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Creates instances covariantly.
	Creates struct {
		CallbackBase
		key any
	}

	// CreatesBuilder builds Creates callbacks.
	CreatesBuilder struct {
		CallbackBuilder
		key any
	}
)

// Creates

func (c *Creates) Key() any {
	return c.key
}

func (c *Creates) Policy() Policy {
	return createsPolicyIns
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

// CreatesBuilder

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
		key:          b.key,
	}
}

// Create creates a value of type parameter T.
func Create[T any](
	handler     Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return CreateKey[T](handler, reflect.TypeFor[T](), constraints...)
}

// CreateKey creates a value of type parameter T with the specified key.
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

// CreateAll creates all values of type parameter T.
func CreateAll[T any](
	handler     Handler,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder CreatesBuilder
	builder.WithKey(reflect.TypeFor[T]()).
		IntoTarget(&t).
		WithConstraints(constraints...)
	creates := builder.New()
	if result := handler.Handle(creates, true, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		if _, p := creates.Result(true); p != nil {
			tp = promise.Return(p, t)
		}
	}
	return
}

var createsPolicyIns Policy = &CovariantPolicy{}
