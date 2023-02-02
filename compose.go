package miruken

type (
	// Composition wraps a callback for composition.
	Composition struct {
		Trampoline
	}

	// CompositionScope decorates a Handler for composition.
	CompositionScope struct {
		Handler
	}
)


// Composition

func (c *Composition) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if cb := c.callback; cb != nil {
		return DispatchCallback(handler, cb, greedy, composer)
	}
	var builder HandlesBuilder
	return builder.WithCallback(c).New().
		Dispatch(handler, greedy, composer)
}

// CompositionScope

func (c *CompositionScope) Handle(
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
