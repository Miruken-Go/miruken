package miruken

import (
	"github.com/miruken-go/miruken/promise"
)

type Trampoline struct {
	callback any
}

func (t *Trampoline) Callback() any {
	return t.callback
}

func (t *Trampoline) Source() any {
	if cb, ok := t.callback.(Callback); ok {
		return cb.Source()
	}
	return nil
}

func (t *Trampoline) Policy() Policy {
	if cb, ok := t.callback.(Callback); ok {
		return cb.Policy()
	}
	return nil
}

func (t *Trampoline) Result(
	many bool,
) (any, *promise.Promise[any]) {
	if cb, ok := t.callback.(Callback); ok {
		return cb.Result(many)
	}
	return nil, nil
}

func (t *Trampoline) SetResult(result any) {
	if cb, ok := t.callback.(Callback); ok {
		cb.SetResult(result)
	}
}

func (t *Trampoline) CanInfer() bool {
	if infer, ok := t.callback.(interface{CanInfer() bool}); ok {
		return infer.CanInfer()
	}
	return true
}

func (t *Trampoline) CanFilter() bool {
	if filter, ok := t.callback.(interface{CanFilter() bool}); ok {
		return filter.CanFilter()
	}
	return true
}

func (t *Trampoline) CanBatch() bool {
	if batch, ok := t.callback.(interface{CanBatch() bool}); ok {
		return batch.CanBatch()
	}
	return true
}

func (t *Trampoline) CanDispatch(
	handler any,
	binding Binding,
) (reset func (), approved bool) {
	if cb := t.callback; cb != nil {
		if guard, ok := cb.(CallbackGuard); ok {
			return guard.CanDispatch(handler, binding)
		}
	}
	return nil, true
}

func (t *Trampoline) Dispatch(
	callback any,
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		panic("callback cannot be nil")
	}
	if cb := t.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, composer)
	}
	var builder HandlesBuilder
	return builder.
		WithCallback(callback).
		NewHandles().Dispatch(handler, greedy, composer)
}