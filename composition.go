package miruken

// composition

type composition struct {
	Trampoline
}

func (c *composition) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if cb := c.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, composer)
	}
	return new(CommandBuilder).
		WithCallback(c).
		NewCommand().
		Dispatch(handler, greedy, composer)
}

// compositionScope

type compositionScope struct {
	Handler
}

func (c *compositionScope) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if composer == nil {
		composer = c
	}
	if _, ok := callback.(composition); !ok {
		callback = &composition{Trampoline{callback}}
	}
	return c.Handler.Handle(callback, greedy, composer)
}
