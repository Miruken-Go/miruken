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
	return mergo.Merge(o.options, options) == nil
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
	if comp, ok := callback.(Composition); ok {
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
		panic(fmt.Sprintf("options: %T is not a *struct or **struct", optType))
	}
	optType = optType.Elem()

	created := false
	options := &options{}

	switch optType.Kind() {
	case reflect.Struct:
		options.options = tv.Interface()
	case reflect.Ptr:
		if optType.Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf("options: %T is not a **struct", optType))
		}
		created = true
		if value := reflect.Indirect(tv); value.IsNil() {
			options.options = reflect.New(optType.Elem()).Interface()
		} else {
			options.options = value.Interface()
		}
	}

	handled := handler.Handle(options, false, nil).IsHandled()
	if handled && created {
		CopyIndirect(options.options, target)
	}
	return handled
}
