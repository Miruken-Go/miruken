package miruken

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// MethodBinder creates a binding a method.
	MethodBinder interface {
		NewMethodBinding(
			method *reflect.Method,
			spec   *bindingSpec,
			key    any,
		) (Binding, error)
	}

	// MethodBinding models a `key` Binding to a method.
	MethodBinding struct {
		funcCall
		BindingBase
		key    any
		method reflect.Method
		lt     reflect.Type
	}

	// MethodBindingError reports a failed method binding.
	MethodBindingError struct {
		Method *reflect.Method
		Cause  error
	}
)

func (b *MethodBinding) Key() any {
	return b.key
}

func (b *MethodBinding) Exported() bool {
	return internal.Exported(b.key) && internal.Exported(b.method)
}

func (b *MethodBinding) LogicalOutputType() reflect.Type {
	return b.lt
}

func (b *MethodBinding) Method() *reflect.Method {
	return &b.method
}

func (b *MethodBinding) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	if initArgs == nil {
		return b.funcCall.Invoke(ctx, ctx.Handler)
	}
	initArgs = append(initArgs, nil)
	copy(initArgs[1:], initArgs)
	initArgs[0] = ctx.Handler
	return b.funcCall.Invoke(ctx, initArgs...)
}

// MethodBindingError

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method %v: %v", e.Method.Name, e.Cause)
}

func (e *MethodBindingError) Unwrap() error {
	return e.Cause
}
