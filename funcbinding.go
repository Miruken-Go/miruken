package miruken

import (
	"fmt"
	"reflect"
)

type (
	// FuncBinder creates a binding to a function.
	FuncBinder interface {
		NewFuncBinding(
			fun  reflect.Value,
			spec *policySpec,
		) (Binding, error)
	}

	// funcBinding models a `key` Binding to a function.
	funcBinding struct {
		FilteredScope
		key      any
		flags    bindingFlags
		fun      reflect.Value
		args     []arg
		metadata []any
	}

	// FuncBindingError reports a failed function binding.
	FuncBindingError struct {
		fun    reflect.Value
		reason error
	}
)

func (b *funcBinding) Key() any {
	return b.key
}

func (b *funcBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *funcBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *funcBinding) Metadata() []any {
	return b.metadata
}

func (b *funcBinding) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	return callFunc(b.fun, context, b.args, explicitArgs...)
}

func (e FuncBindingError) Func() reflect.Value {
	return e.fun
}

func (e FuncBindingError) Error() string {
	return fmt.Sprintf("invalid function %v: %v", e.fun, e.reason)
}

func (e FuncBindingError) Unwrap() error { return e.reason }

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
		// handlers args are always passed first
		res := fun.Call(append(args, deps...))
		results := make([]any, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results, nil
	}
}