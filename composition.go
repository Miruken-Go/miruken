package miruken

// Composition

type Composition struct {
	Trampoline
}

func (c *Composition) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if cb := c.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, composer)
	}
	var builder HandlesBuilder
	return builder.
		WithCallback(c).
		NewHandles().
		Dispatch(handler, greedy, composer)
}

// compositionScope

type compositionScope struct {
	Handler
}

func (c *compositionScope) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if composer == nil {
		composer = c
	}
	if _, ok := callback.(*Composition); !ok {
		callback = &Composition{Trampoline{callback}}
	}
	return c.Handler.Handle(callback, greedy, composer)
}
