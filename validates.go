package miruken

import "reflect"

// Validates callbacks contravariantly.
type Validates struct {
	CallbackBase
	target interface{}
}

func (v *Validates) Target() interface{} {
	return v.target
}

func (v *Validates) Key() interface{} {
	return reflect.TypeOf(v.target)
}

func (v *Validates) Policy() Policy {
	return _validatesPolicy
}

func (v *Validates) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, v.target, v, greedy, composer)
}

var _validatesPolicy Policy = &ContravariantPolicy{}
