package miruken

import (
	"fmt"
	"reflect"
)

// ConstructorBinder creates a constructor binding to `handlerType`.
type ConstructorBinder interface {
	NewConstructorBinding(
		handlerType  reflect.Type,
		constructor *reflect.Method,
		spec        *policySpec,
	) (Binding, error)
}

// constructorBinding models the creation/initialization
// of the `handlerType`.
type constructorBinding struct {
	FilteredScope
	handlerType  reflect.Type
	flags        bindingFlags
}

func (b *constructorBinding) Key() any {
	return b.handlerType
}

func (b *constructorBinding) Strict() bool {
	return false
}

func (b *constructorBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *constructorBinding) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	if len(explicitArgs) > 0 {
		return nil, nil  // return nothing if not called as constructor
	}
	var handler any
	handlerType := b.handlerType
	if handlerType.Kind() == reflect.Ptr {
		handler = reflect.New(handlerType.Elem()).Interface()
	} else {
		handler = reflect.New(handlerType).Elem().Interface()
	}
	return []any{handler}, nil
}

func newConstructorBinding(
	handlerType   reflect.Type,
	constructor  *reflect.Method,
	spec         *policySpec,
	explicitSpec  bool,
) (binding *constructorBinding, invalid error) {
	binding = &constructorBinding{
		handlerType: handlerType,
	}
	if spec != nil {
		binding.providers = spec.filters
		binding.flags     = spec.flags
	}
	if constructor != nil {
		startIndex := 0
		methodType := constructor.Type
		numArgs    := methodType.NumIn()
		args       := make([]arg, numArgs-1)  // skip receiver
		if spec != nil && explicitSpec {
			startIndex = 1
			args[0] = zeroArg{} // policy/binding placeholder
		}
		if err := buildDependencies(methodType, startIndex+1, numArgs, args, startIndex); err != nil {
			invalid = fmt.Errorf("constructor: %w", err)
		} else {
			initializer := &initializer{*constructor, args}
			binding.AddFilters(&initializerProvider{[]Filter{initializer}})
		}
	}
	return binding, invalid
}
