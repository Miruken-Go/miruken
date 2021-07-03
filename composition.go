package miruken

// Composition

type Composition struct {
	Trampoline
}

func (c *Composition) Dispatch(
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
	if _, ok := callback.(Composition); !ok {
		callback = &Composition{Trampoline{callback}}
	}
	return c.Handler.Handle(callback, greedy, composer)
}
