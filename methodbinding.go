package miruken

import (
	"fmt"
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
		key    any
		flags  bindingFlags
		method reflect.Method
		args   []arg
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

func (b *methodBinding) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	return callFunc(b.method.Func, context, b.args, explicitArgs...)
}

func (e MethodBindingError) Method() reflect.Method {
	return e.method
}

func (e MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method %v: %v", e.method.Name, e.reason)
}

func (e MethodBindingError) Unwrap() error { return e.reason }
