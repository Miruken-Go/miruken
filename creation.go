package miruken

import "reflect"

type Creation struct {
	CallbackBase
	typ reflect.Type
}

func (c *Creation) Type() interface{} {
	return c.typ
}

func (c *Creation) Policy() Policy {
	return CreatesPolicy()
}

func (c *Creation) ReceiveResult(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	if result == nil {
		return false
	}
	c.results = append(c.results, result)
	c.result  = nil
	return true
}

func (c *Creation) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(c.Policy(), handler, c, c, c.typ, greedy, composer, c).
		OtherwiseHandledIf(len(c.results) > count)
}

type CreationBuilder struct {
	CallbackBuilder
	typ reflect.Type
}

func (b *CreationBuilder) WithType(
	typ reflect.Type,
) *CreationBuilder {
	b.typ = typ
	return b
}

func (b *CreationBuilder) NewCreation() *Creation {
	return &Creation{
		CallbackBase: b.Callback(),
		typ: b.typ,
	}
}

func Create(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv     := TargetValue(target)
	create := new(CreationBuilder).
		WithType(tv.Type().Elem()).
		NewCreation()
	if result := handler.Handle(create, false, nil); result.IsError() {
		return result.Error()
	}
	create.CopyResult(tv)
	return nil
}

func CreateAll(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	builder := new(CreationBuilder).
		WithType(tv.Type().Elem().Elem())
	builder.WithMany()
	create := builder.NewCreation()
	if result := handler.Handle(create, true, nil); result.IsError() {
		return result.Error()
	}
	create.CopyResult(tv)
	return nil
}