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
		) (Binding, error)
	}

	// methodBinding models a `key` Binding to a method.
	methodBinding struct {
		FilteredScope
		key      any
		flags    bindingFlags
		method   reflect.Method
		args     []arg
		metadata []any
	}

	// MethodBindingError reports a failed method binding.
	MethodBindingError struct {
		method reflect.Method
		reason error
	}
)

func (b *methodBinding) Key() any {
	return b.key
}

func (b *methodBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *methodBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *methodBinding) Metadata() []any {
	return b.metadata
}

func (b *methodBinding) Invoke(
	ctx      HandleContext,
	initArgs ... any,
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

func (b *methodBinding) Method() reflect.Method {
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
