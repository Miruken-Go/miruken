package miruken

import (
	"reflect"
)

// Handles callbacks contravariantly.
type Handles struct {
	CallbackBase
	callback any
}

func (h *Handles) Callback() any {
	return h.callback
}

func (h *Handles) Key() any {
	return reflect.TypeOf(h.callback)
}

func (h *Handles) Policy() Policy {
	return _handlesPolicy
}

func (h *Handles) CanDispatch(
	handler     any,
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
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	count := len(h.results)
	return DispatchPolicy(handler, h.callback, h, greedy, composer).
		OtherwiseHandledIf(len(h.results) > count)
}

// HandlesBuilder builds Handles callbacks.
type HandlesBuilder struct {
	CallbackBuilder
	callback any
}

func (b *HandlesBuilder) WithCallback(
	callback any,
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

func Invoke(handler Handler, callback any, target any) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv     := TargetValue(target)
	var builder HandlesBuilder
	handle := builder.
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

func InvokeAll(handler Handler, callback any, target any) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	var builder HandlesBuilder
	builder.WithCallback(callback).WithMany()
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
