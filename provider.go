package miruken

import "reflect"

type provider struct {
	value interface{}
	typ   reflect.Type
}

func (p *provider) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.Callback()
	}
	if inquiry, ok := callback.(*Provides); ok {
		if typ, ok := inquiry.key.(reflect.Type); ok {
			if p.typ.AssignableTo(typ) {
				return NotHandled.OtherwiseHandledIf(
					inquiry.ReceiveResult(p.value, true, greedy, composer))
			}
		}
	}
	return NotHandled
}

func (p *provider) suppressDispatch() {}

func NewProvider(value interface{}) Handler {
	if value == nil {
		panic("value cannot be nil")
	}
	return &provider{
		value: value,
		typ:   reflect.TypeOf(value),
	}
}