package miruken

// composition

type composition struct {
	Trampoline
}

func (c *composition) Dispatch(
	handler interface{},
	greedy  bool,
	ctx     HandleContext,
) HandleResult {
	if cb := c.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, ctx)
	}
	return (&Command{callback: c}).Dispatch(handler, greedy, ctx)
}

// compositionScope

type compositionScope struct {
	HandleContext
}

func (c *compositionScope) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	if ctx == nil {
		ctx = c
	}
	if _, ok := callback.(composition); !ok {
		callback = &composition{Trampoline{callback}}
	}
	return c.HandleContext.Handle(callback, greedy, ctx)
}
