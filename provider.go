package miruken

import "reflect"

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
	if provides, ok := callback.(*Provides); ok {
		if typ, ok := provides.key.(reflect.Type); ok {
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
	return &provider{
		value: value,
		typ:   reflect.TypeOf(value),
	}
}