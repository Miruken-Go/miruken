package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// MethodBinder creates a binding a method.
	MethodBinder interface {
		NewMethodBinding(
			method reflect.Method,
			spec   *policySpec,
			key    any,
		) (Binding, error)
	}

	// MethodBinding models a `key` Binding to a method.
	MethodBinding struct {
		BindingBase
		key    any
		method reflect.Method
		args   []arg
	}

	// MethodBindingError reports a failed method binding.
	MethodBindingError struct {
		method reflect.Method
		reason error
	}
)

func (b *MethodBinding) Key() any {
	return b.key
}

func (b *MethodBinding) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	if initArgs == nil {
		initArgs = []any{ctx.handler}
	} else {
		initArgs = append(initArgs, nil)
		copy(initArgs[1:], initArgs)
		initArgs[0] = ctx.handler
	}
	return callFunc(b.method.Func, ctx, b.args, initArgs...)
}

func (b *MethodBinding) Method() reflect.Method {
	return b.method
}

// MethodBindingError

func (e *MethodBindingError) Method() reflect.Method {
	return e.method
}

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method %v: %v", e.method.Name, e.reason)
}

func (e *MethodBindingError) Unwrap() error {
	return e.reason
}
