package miruken

import "reflect"

type Trampoline struct {
	callback interface{}
}

func (t *Trampoline) Callback() interface{} {
	return t.callback
}

func (t *Trampoline) ResultType() reflect.Type {
	if cb, ok := t.callback.(Callback); ok {
		return cb.ResultType()
	}
	return nil
}

func (t *Trampoline) Result() interface{} {
	if cb, ok := t.callback.(Callback); ok {
		return cb.Result()
	}
	return nil
}

func (t *Trampoline) SetResult(result interface{}) {
	if cb, ok := t.callback.(Callback); ok {
		cb.SetResult(result)
	}
}

func (t *Trampoline) Policy() Policy {
	if cb, ok := t.callback.(Callback); ok {
		return cb.Policy()
	}
	return nil
}

func (t *Trampoline) CanInfer() bool {
	if infer, ok := t.callback.(interface{CanInfer() bool}); ok {
		return infer.CanInfer()
	}
	return true
}

func (t *Trampoline) CanFilter() bool {
	if infer, ok := t.callback.(interface{CanFilter() bool}); ok {
		return infer.CanFilter()
	}
	return true
}

func (t *Trampoline) CanDispatch(
	handler interface{},
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
	callback interface{},
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		panic("callback cannot be nil")
	}
	if cb := t.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, composer)
	}
	return new(CommandBuilder).
		WithCallback(callback).
		NewCommand().Dispatch(handler, greedy, composer)
}