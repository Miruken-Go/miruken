package miruken

import (
	"reflect"
)

type provider struct {
	value any
	typ   reflect.Type
}

func (p *provider) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.Callback()
	}
	if provides, ok := callback.(* Provides); ok {
		if typ, ok := provides.Key().(reflect.Type); ok {
			if p.typ.AssignableTo(typ) {
				return provides.ReceiveResult(p.value, true, composer)
			}
		}
	}
	return NotHandled
}

func (p *provider) SuppressDispatch() {}


func NewProvider(value any) Handler {
	if value == nil {
		panic("value cannot be nil")
	}
	return &provider{value: value, typ:reflect.TypeOf(value)}
}

func With(values ...any) BuilderFunc {
	return func (handler Handler) Handler {
		var valueHandlers []any
		for _, val := range values {
			if val != nil {
				valueHandlers = append(valueHandlers, NewProvider(val))
			}
		}
		if len(valueHandlers) > 0 {
			return AddHandlers(handler, valueHandlers...)
		}
		return handler
	}
}