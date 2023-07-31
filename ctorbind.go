package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// ConstructorBinder creates a constructor binding to `handlerType`.
	ConstructorBinder interface {
		NewConstructorBinding(
			handlerType  reflect.Type,
			constructor *reflect.Method,
			spec        *bindingSpec,
			key         any,
		) (Binding, error)
	}

	// ConstructorBinding customizes the construction of `handlerType`.
	ConstructorBinding struct {
		BindingBase
		handlerType reflect.Type
		key         any
	}
)


func (b *ConstructorBinding) Key() any {
	if key := b.key; key != nil {
		return key
	}
	return b.handlerType
}

func (b *ConstructorBinding) Strict() bool {
	return false
}

func (b *ConstructorBinding) Exported() bool {
	return false
}

func (b *ConstructorBinding) LogicalOutputType() reflect.Type {
	return b.handlerType
}

func (b *ConstructorBinding) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	// ConstructorBinding's will be called on existing
	// handlers if present.  This would result in an
	// additional and unexpected instance created.
	// This situation can be detected if the handler is
	// the same type created by this binding.  If it is,
	// the creation will be skipped.  Otherwise, a true
	// construction is desired.
	handlerType := b.handlerType
	if reflect.TypeOf(ctx.Handler) == handlerType {
		return nil, nil, nil
	}
	var receiver any
	if handlerType.Kind() == reflect.Ptr {
		receiver = reflect.New(handlerType.Elem()).Interface()
	} else {
		receiver = reflect.New(handlerType).Elem().Interface()
	}
	return []any{receiver}, nil, nil
}

func newConstructorBinding(
	handlerType  reflect.Type,
	constructor  *reflect.Method,
	spec         *bindingSpec,
	key          any,
	explicitSpec bool,
) (binding *ConstructorBinding, err error) {
	binding = &ConstructorBinding{
		BindingBase{
			FilteredScope{spec.filters},
			spec.flags,
			spec.metadata,
		},
		handlerType,
		key,
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
		if err = buildDependencies(methodType, startIndex+1, numArgs, args, startIndex); err != nil {
			err = fmt.Errorf("constructor: %w", err)
		} else {
			initializer := &initializer{*constructor, args}
			binding.AddFilters(&initializerProvider{[]Filter{initializer}})
		}
	}
	return binding, err
}
