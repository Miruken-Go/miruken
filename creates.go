package miruken

import (
	"reflect"
)

// Creates instances Covariantly.
type Creates struct {
	CallbackBase
	typ reflect.Type
}

func (c *Creates) Key() interface{} {
	return c.typ
}

func (c *Creates) Policy() Policy {
	return _createsPolicy
}

func (c *Creates) ReceiveResult(
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

func (c *Creates) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(handler, c, c, greedy, composer, c).
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

func (b *CreationBuilder) NewCreation() *Creates {
	return &Creates{
		CallbackBase: b.CallbackBase(),
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

// createsPolicy for creating instances covariantly.
type createsPolicy struct {
	CovariantPolicy
}

func (c *createsPolicy) NewConstructorBinding(
	handlerType  reflect.Type,
	constructor *reflect.Method,
	spec        *policySpec,
) (binding Binding, err error) {
	return newConstructorBinding(handlerType, constructor, spec, spec != nil)
}

var _createsPolicy Policy = &createsPolicy{}