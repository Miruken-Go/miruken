package provides

import (
	"github.com/miruken-go/miruken"
	"reflect"
)

type provider struct {
	value any
	typ   reflect.Type
}

func (p *provider) Handle(
	callback any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if comp, ok := callback.(*miruken.Composition); ok {
		callback = comp.Callback()
	}
	if provides, ok := callback.(*It); ok {
		if typ, ok := provides.Key().(reflect.Type); ok {
			if p.typ.AssignableTo(typ) {
				return provides.ReceiveResult(p.value, true, composer)
			}
		}
	}
	return miruken.NotHandled
}

func (p *provider) SuppressDispatch() {}


func NewProvider(value any) miruken.Handler {
	if value == nil {
		panic("value cannot be nil")
	}
	return &provider{value: value, typ:reflect.TypeOf(value)}
}

func With(values ...any) miruken.BuilderFunc {
	return func (handler miruken.Handler) miruken.Handler {
		var valueHandlers []any
		for _, val := range values {
			if val != nil {
				valueHandlers = append(valueHandlers, NewProvider(val))
			}
		}
		if len(valueHandlers) > 0 {
			return miruken.AddHandlers(handler, valueHandlers...)
		}
		return handler
	}
}