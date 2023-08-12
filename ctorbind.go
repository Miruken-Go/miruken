package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// ConstructorBinder creates a constructor binding to `handlerType`.
	ConstructorBinder interface {
		NewCtorBinding(
			typ  reflect.Type,
			ctor *reflect.Method,
			spec *bindingSpec,
			key  any,
		) (Binding, error)
	}

	// ctorBinding customizes the construction of `handlerType`.
	ctorBinding struct {
		BindingBase
		handlerType reflect.Type
		key         any
	}
)


func (b *ctorBinding) Key() any {
	if key := b.key; key != nil {
		return key
	}
	return b.handlerType
}

func (b *ctorBinding) Strict() bool {
	return false
}

func (b *ctorBinding) Exported() bool {
	return false
}

func (b *ctorBinding) LogicalOutputType() reflect.Type {
	return b.handlerType
}

func (b *ctorBinding) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	// ctorBinding's will be called on existing
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

func newCtorBinding(
	typ          reflect.Type,
	ctor         *reflect.Method,
	spec         *bindingSpec,
	key          any,
	explicitSpec bool,
) (binding *ctorBinding, err error) {
	binding = &ctorBinding{
		BindingBase{
			FilteredScope{spec.filters},
			spec.flags,
			spec.metadata,
		},
		typ,
		key,
	}
	if ctor != nil {
		startIndex := 0
		methodType := ctor.Type
		numArgs    := methodType.NumIn()
		args       := make([]arg, numArgs-1)  // skip receiver
		if spec != nil && explicitSpec {
			startIndex = 1
			args[0] = zeroArg{} // policy/binding placeholder
		}
		if err = buildDependencies(methodType, startIndex+1, numArgs, args, startIndex); err != nil {
			err = fmt.Errorf("ctor: %w", err)
		} else {
			initializer := &initializer{*ctor, args}
			binding.AddFilters(&initProvider{[]Filter{initializer}})
		}
	}
	return binding, err
}
