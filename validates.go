package miruken

import "reflect"

// Validates callbacks contravariantly.
type Validates struct {
	CallbackBase
	target any
}

func (v *Validates) Target() any {
	return v.target
}

func (v *Validates) Key() any {
	return reflect.TypeOf(v.target)
}

func (v *Validates) Policy() Policy {
	return _validatesPolicy
}

func (v *Validates) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, v.target, v, greedy, composer)
}

var _validatesPolicy Policy = &ContravariantPolicy{}
