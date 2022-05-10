package miruken

import (
	"fmt"
	"reflect"
)

// FuncBindingError reports a failed function binding.
type FuncBindingError struct {
	Func   reflect.Value
	Reason error
}

func (e FuncBindingError) Error() string {
	return fmt.Sprintf("invalid function: %#v: %v", e.Func, e.Reason)
}

func (e FuncBindingError) Unwrap() error { return e.Reason }

// FuncBinder creates a binding to a function.
type FuncBinder interface {
	NewFuncBinding(
		fun  reflect.Value,
		spec *policySpec,
	) (Binding, error)
}

// funcBinding models a `key` Binding to a function.
type funcBinding struct {
	FilteredScope
	key   any
	flags bindingFlags
	fun   reflect.Value
	args  []arg
}

func (b *funcBinding) Key() any {
	return b.key
}

func (b *funcBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *funcBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *funcBinding) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	return callFunc(b.fun, context, b.args, explicitArgs...)
}

func callFunc(
	fun          reflect.Value,
	context      HandleContext,
	dependencies []arg,
	explicitArgs ... any,
) ([]any, error) {
	fromIndex := len(explicitArgs)
	if deps, err := resolveArgs(fun.Type(), fromIndex, dependencies, context); err != nil {
		return nil, err
	} else {
		var args []reflect.Value
		for _, arg := range explicitArgs {
			args = append(args, reflect.ValueOf(arg))
		}
		// explicit args are always passed first
		res := fun.Call(append(args, deps...))
		results := make([]any, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results, nil
	}
}