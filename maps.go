package miruken

import "reflect"

// Maps callbacks Bivariantly.
type Maps struct {
	CallbackBase
	source       interface{}
	typeOrTarget interface{}
	format       interface{}
}

func (m *Maps) Source() interface{} {
	return m.source
}

func (m *Maps) TypeOrTarget() interface{} {
	return m.typeOrTarget
}

func (m *Maps) Format() interface{} {
	return m.format
}

func (m *Maps) Key() interface{} {
	in := reflect.TypeOf(m.source)
	switch out := m.typeOrTarget.(type) {
	case reflect.Type:
		return DiKey{In: in, Out: out}
	default:
		return DiKey{In: in, Out: reflect.TypeOf(out)}
	}
}

func (m *Maps) Policy() Policy {
	return _mapsPolicy
}

func (m *Maps) ReceiveResult(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return m.AddResult(result)
}

func (m *Maps) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, m.source, m, greedy, composer, m).
		OtherwiseHandledIf(m.Result() != nil)
}

var _mapsPolicy Policy = &BivariantPolicy{}