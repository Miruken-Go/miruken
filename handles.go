package miruken

import (
	"reflect"
)

// Handles callbacks Covariantly.
type Handles struct {
	CallbackBase
	callback interface{}
}

func (h *Handles) Callback() interface{} {
	return h.callback
}

func (h *Handles) Key() interface{} {
	return reflect.TypeOf(h.callback)
}

func (h *Handles) Policy() Policy {
	return _handlesPolicy
}

func (h *Handles) ReceiveResult(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return h.AddResult(result)
}

func (h *Handles) CanDispatch(
	handler     interface{},
	binding     Binding,
) (reset func (), approved bool) {
	if guard, ok := h.callback.(CallbackGuard); ok {
		return guard.CanDispatch(handler, binding)
	}
	return nil, true
}

func (h *Handles) CanInfer() bool {
	if infer, ok := h.callback.(interface{CanInfer() bool}); ok {
		return infer.CanInfer()
	}
	return true
}

func (h *Handles) CanFilter() bool {
	if infer, ok := h.callback.(interface{CanFilter() bool}); ok {
		return infer.CanFilter()
	}
	return true
}

func (h *Handles) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	count := len(h.results)
	return DispatchPolicy(handler, h.callback, h, greedy, composer, h).
		OtherwiseHandledIf(len(h.results) > count)
}

// HandlesBuilder builds Handles callbacks.
type HandlesBuilder struct {
	CallbackBuilder
	callback interface{}
}

func (b *HandlesBuilder) WithCallback(
	callback interface{},
) *HandlesBuilder {
	if IsNil(callback) {
		panic("callback cannot be nil")
	}
	b.callback = callback
	return b
}

func (b *HandlesBuilder) NewHandles() *Handles {
	return &Handles{
		CallbackBase: b.CallbackBase(),
		callback:     b.callback,
	}
}

func Invoke(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv     := TargetValue(target)
	handle := new(HandlesBuilder).
		WithCallback(callback).
		NewHandles()
	if result := handler.Handle(handle, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return NotHandledError{callback}
	}
	handle.CopyResult(tv)
	return nil
}

func InvokeAll(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	builder := new(HandlesBuilder).WithCallback(callback)
	builder.WithMany()
	handle  := builder.NewHandles()
	if result := handler.Handle(handle, true, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return NotHandledError{callback}
	}
	handle.CopyResult(tv)
	return nil
}

var _handlesPolicy Policy = &ContravariantPolicy{}
