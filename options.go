package miruken

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/miruken-go/miruken/promise"
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
	options any
}

func (o *options) CanInfer() bool {
	return false
}

func (o *options) CanFilter() bool {
	return false
}

func (o *options) mergeFrom(options any) bool {
	return MergeOptions(options, o.options)
}

// optionsHandler merges compatible options.
type optionsHandler struct {
	Handler
	options any
	optionsType reflect.Type
}

func (c *optionsHandler) Handle(
	callback any,
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
				MergeFrom(options any) bool
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

func MergeOptions(from, into any) bool {
	return mergo.Merge(into, from, mergo.WithAppendSlice) == nil
}

func Options(options any) BuilderFunc {
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
	return func (handler Handler) Handler {
		return &optionsHandler{handler, options, optType}
	}
}

func GetOptions(handler Handler, target any) bool {
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
	typ reflect.Type,
	dep DependencyArg,
	ctx HandleContext,
) (options reflect.Value, _ *promise.Promise[reflect.Value], err error) {
	options = reflect.New(typ)
	if GetOptions(ctx.composer, options.Interface()) {
		if typ.Kind() == reflect.Ptr {
			return options, nil, nil
		}
		return reflect.Indirect(options), nil, nil
	}
	if dep.Optional() {
		return reflect.Zero(typ), nil, nil
	}
	var v reflect.Value
	return v, nil, fmt.Errorf("FromOptions: unable to resolve options %v", typ)
}