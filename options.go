package miruken

import (
	"fmt"
	"github.com/imdario/mergo"
	"reflect"
)

// Options represent extensible settings.
type options struct{
	options interface{}
}

func (o *options) CanInfer() bool {
	return false
}

func (o *options) mergeFrom(options interface{}) bool {
	return MergeOptions(options, o.options)
}

// optionsHandler merges compatible options.
type optionsHandler struct {
	options interface{}
	optionsType reflect.Type
}

func (c *optionsHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	if comp, ok := callback.(*Composition); ok {
		if callback = comp.Callback(); callback == nil {
			return NotHandled
		}
	}
	if opt, ok := callback.(*options); ok {
		options := opt.options
		if reflect.TypeOf(options).Elem().AssignableTo(c.optionsType) {
			if o, ok := options.(interface {
				MergeFrom(options interface{}) bool
			}); ok {
				if o.MergeFrom(c.options) {
					return Handled
				}
			} else {
				if opt.mergeFrom(c.options) {
					return Handled
				}
			}
		}
	}
	return NotHandled
}

func MergeOptions(from, into interface{}) bool {
	return mergo.Merge(into, from, mergo.WithAppendSlice) == nil
}

func WithOptions(options interface{}) Builder {
	optType := reflect.TypeOf(options)
	if optType == nil {
		panic("options cannot be nil")
	}
	if optType.Kind() == reflect.Ptr {
		optType = optType.Elem()
	}
	if optType.Kind() != reflect.Struct {
		panic("options must be a struct or *struct")
	}
	return BuilderFunc(func (handler Handler) Handler {
		return AddHandlers(handler, &optionsHandler{options, optType})
	})
}

func GetOptions(handler Handler, target interface{}) bool {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv := TargetValue(target)
	optType := tv.Type()
	if optType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("options: %v is not a *struct or **struct", optType))
	}
	optType = optType.Elem()

	created := false
	options := &options{}

	switch optType.Kind() {
	case reflect.Struct:
		options.options = tv.Interface()
	case reflect.Ptr:
		if optType.Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf("options: %v is not a **struct", optType))
		}
		created = true
		if value := reflect.Indirect(tv); value.IsNil() {
			options.options = reflect.New(optType.Elem()).Interface()
		} else {
			options.options = value.Interface()
		}
	}

	handled := handler.Handle(options, true, nil).IsHandled()
	if handled && created {
		CopyIndirect(options.options, target)
	}
	return handled
}

// FromOptions is a DependencyResolver that binds the argument to options.
type FromOptions struct {}

func (o FromOptions) Validate(
	typ  reflect.Type,
	dep *dependencyArg,
) error {
	optType := dep.ArgType(typ)
	if optType.Kind() == reflect.Ptr {
		optType = optType.Elem()
	}
	if optType.Kind() != reflect.Struct {
		return fmt.Errorf("FromOptions: %v is not a struct or *struct", optType)
	}
	return nil
}

func (o FromOptions) Resolve(
	typ          reflect.Type,
	rawCallback  interface{},
	dep         *dependencyArg,
	handler      Handler,
) (options reflect.Value, err error) {
	optType := dep.ArgType(typ.Elem())
	options = reflect.New(optType)
	if !GetOptions(handler, options.Interface()) {
		return reflect.ValueOf(nil), fmt.Errorf(
			"FromOptions: unable to resolve options %v", optType)
	}
	if optType.Kind() == reflect.Ptr {
		return options, nil
	}
	return reflect.Indirect(options), nil
}