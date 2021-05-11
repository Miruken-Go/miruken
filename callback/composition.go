package callback

// composition

type composition struct {
	Trampoline
}

func (c *composition) Dispatch(
	handler  interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if cb := c.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, context)
	}
	return (&Command{callback: c}).Dispatch(handler, greedy, context)
}

// compositionScope

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
		callback = &composition{Trampoline{callback}}
	}
	return c.HandleContext.Handle(callback, greedy, context)
}
