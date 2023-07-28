package miruken

type (
	// Resolves extends Provides semantics to dispatch a
	// Callback on the resolved handler.
	Resolves struct {
		Provides
		callback  Callback
		greedy    bool
		succeeded bool
	}

	// ResolvesBuilder builds Resolves callbacks.
	ResolvesBuilder struct {
		ProvidesBuilder
		callback Callback
		greedy   bool
	}
)


// Resolves

func (r *Resolves) Callback() Callback {
	return r.callback
}

func (r *Resolves) Succeeded() bool {
	return r.succeeded
}

func (r *Resolves) CanDispatch(
	handler any,
	binding Binding,
) (reset func (), approved bool) {
	if outer, ok := r.Provides.CanDispatch(handler, binding); !ok {
		return outer, false
	} else if guard, ok := r.callback.(CallbackGuard); !ok {
		return outer, true
	} else if inner, ok := guard.CanDispatch(handler, binding); !ok {
		outer()
		return inner, false
	} else {
		return func() {
			inner()
			outer()
		}, true
	}
}

func (r *Resolves) accept(
	result   any,
	composer Handler,
) HandleResult {
	if greedy := r.greedy; !greedy && r.succeeded {
		return Handled
	} else {
		hr := DispatchCallback(result, r.callback, greedy, composer)
		r.succeeded = r.succeeded || hr.handled
		return hr
	}
}


// ResolvesBuilder

func (b *ResolvesBuilder) WithCallback(
	callback Callback,
) *ResolvesBuilder {
	b.callback = callback
	return b
}

func (b *ResolvesBuilder) WithGreedy(
	greedy bool,
) *ResolvesBuilder {
	b.greedy = greedy
	return b
}

func (b *ResolvesBuilder) New() *Resolves {
	resolves := &Resolves{
		Provides: b.Build(),
		callback: b.callback,
		greedy:   b.greedy,
	}
	resolves.SetAcceptResult(resolves.accept)
	return resolves
}
