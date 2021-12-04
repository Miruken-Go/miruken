package miruken

import (
	"fmt"
	"github.com/imdario/mergo"
	"reflect"
)

// OptionBool should be used in option structs instead of bool to
// be able to represent a bool not set.  Otherwise, the Zero value
// for of a bool cannot be distinguished from false.
type OptionBool byte
const (
	OptionNone OptionBool = iota
	OptionFalse
	OptionTrue
)

func (b OptionBool) Bool() bool {
	switch b {
	case OptionFalse: return false
	case OptionTrue: return true
	default:
		panic("only OptionFalse and OptionTrue can convert to a bool")
	}
}

// Options represent extensible settings.
type options struct {
	options interface{}
}

func (o *options) CanInfer() bool {
	return false
}

func (o *options) CanFilter() bool {
	return false
}

func (o *options) mergeFrom(options interface{}) bool {
	return MergeOptions(options, o.options)
}

// optionsHandler merges compatible options.
type optionsHandler struct {
	Handler
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
	tryInitializeComposer(&composer, c)
	if comp, ok := callback.(*Composition); ok {
		if callback = comp.Callback(); callback == nil {
			return c.Handler.Handle(callback, greedy, composer)
		}
	}
	if opt, ok := callback.(*options); ok {
		options := opt.options
		if reflect.TypeOf(options).Elem().AssignableTo(c.optionsType) {
			merged := false
			if o, ok := options.(interface {
				MergeFrom(options interface{}) bool
			}); ok {
				merged = o.MergeFrom(c.options)
			} else {
				merged = opt.mergeFrom(c.options)
			}
			if merged {
				if greedy {
					return c.Handler.Handle(callback, greedy, composer).Or(Handled)
				}
				return Handled
			}
		}
	}
	return c.Handler.Handle(callback, greedy, composer)
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
		return &optionsHandler{handler, options, optType}
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

// FromOptions is a DependencyResolver that binds options to an argument.
type FromOptions struct {}

func (o FromOptions) Validate(
	typ reflect.Type,
	dep DependencyArg,
) error {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("FromOptions: %v is not a struct or *struct", typ)
	}
	return nil
}

func (o FromOptions) Resolve(
	typ         reflect.Type,
	rawCallback Callback,
	dep         DependencyArg,
	handler     Handler,
) (options reflect.Value, err error) {
	options = reflect.New(typ)
	if GetOptions(handler, options.Interface()) {
		if typ.Kind() == reflect.Ptr {
			return options, nil
		}
		return reflect.Indirect(options), nil
	}
	if dep.Optional() {
		return reflect.Zero(typ), nil
	}
	return reflect.ValueOf(nil), fmt.Errorf(
		"FromOptions: unable to resolve options %v", typ)
}