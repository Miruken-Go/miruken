package miruken

import (
	"fmt"
	"reflect"
)

type (
	// ConstructorBinder creates a constructor binding to `handlerType`.
	ConstructorBinder interface {
		NewConstructorBinding(
			handlerType  reflect.Type,
			constructor *reflect.Method,
			spec        *policySpec,
		) (Binding, error)
	}

	// constructorBinding customizes the construction of `handlerType`.
	constructorBinding struct {
		FilteredScope
		handlerType  reflect.Type
		flags        bindingFlags
	}
)

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
	// constructorBinding's will be called on existing
	// handlers if present.  This would result in an
	// additional and unexpected instance created.
	// This situation can be detected if the handler is
	// the same type created by this binding.  If it is,
	// the creation will be skipped.  Otherwise, a true
	// construction is desired.
	handlerType := b.handlerType
	if reflect.TypeOf(context.handler) == handlerType {
		return nil, nil
	}
	var receiver any
	if handlerType.Kind() == reflect.Ptr {
		receiver = reflect.New(handlerType.Elem()).Interface()
	} else {
		receiver = reflect.New(handlerType).Elem().Interface()
	}
	return []any{receiver}, nil
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
