package callback

import "reflect"

type Trampoline struct {
	callback interface{}
}

func (t *Trampoline) GetCallback() interface{} {
	return t.callback
}

func (t *Trampoline) GetResultType() reflect.Type {
	if cb, ok := t.callback.(Callback); ok {
		return cb.GetResultType()
	}
	return nil
}

func (t *Trampoline) GetResult() interface{} {
	if cb, ok := t.callback.(Callback); ok {
		return cb.GetResult()
	}
	return nil
}

func (t *Trampoline) SetResult(result interface{}) {
	if cb, ok := t.callback.(Callback); ok {
		cb.SetResult(result)
	}
}

func (t *Trampoline) GetPolicy() Policy {
	if cb, ok := t.callback.(CallbackDispatcher); ok {
		return cb.GetPolicy()
	}
	return nil
}

func (t *Trampoline) DispatchTrampoline(
	callback interface{},
	handler  interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if callback == nil {
		panic("nil callback")
	}
	if cb := t.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, context)
	}
	command := &Command{callback: callback}
	return command.Dispatch(handler, greedy, context)
}