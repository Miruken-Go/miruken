package callback

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
	if cb, ok := t.callback.(CallbackDispatcher); ok {
		return cb.Policy()
	}
	return nil
}

func (t *Trampoline) DispatchTrampoline(
	callback interface{},
	handler  interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	if callback == nil {
		panic("nil callback")
	}
	if cb := t.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, ctx)
	}
	command := &Command{callback: callback}
	return command.Dispatch(handler, greedy, ctx)
}