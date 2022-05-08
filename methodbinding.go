package miruken

import (
	"fmt"
	"reflect"
)

// MethodBindingError reports a failed method binding.
type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

func (e MethodBindingError) Unwrap() error { return e.Reason }

// MethodBinder creates a binding a method.
type MethodBinder interface {
	NewMethodBinding(
		method reflect.Method,
		spec   *policySpec,
	) (Binding, error)
}

// methodBinding models a `key` Binding to a method.
type methodBinding struct {
	FilteredScope
	key    any
	flags  bindingFlags
	method reflect.Method
	args   []arg
}

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