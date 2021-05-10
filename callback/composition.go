package callback

import "reflect"

type composition struct {
	Callback interface{}
}

func (c *composition) GetPolicy() Policy {
	if cb, ok := c.Callback.(CallbackDispatcher); ok {
		return cb.GetPolicy()
	}
	return nil
}

func (c *composition) GetResultType() reflect.Type {
	if cb, ok := c.Callback.(Callback); ok {
		return cb.GetResultType()
	}
	return nil
}

func (c *composition) GetResult() interface{} {
	if cb, ok := c.Callback.(Callback); ok {
		return cb.GetResult()
	}
	return nil
}

func (c *composition) SetResult(result interface{}) {
	if cb, ok := c.Callback.(Callback); ok {
		cb.SetResult(result)
	}
}

func (c *composition) Dispatch(
	handler  interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if cb := c.Callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, context)
	}
	command := &Command{Callback: c}
	return command.Dispatch(handler, greedy, context)
}

type compositionScope struct {
	HandleContext
}

func (c *compositionScope) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = c
	}
	if _, ok := callback.(composition); !ok {
		callback = composition{callback}
	}
	return c.HandleContext.Handle(callback, greedy, context)
}

