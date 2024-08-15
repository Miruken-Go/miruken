package miruken

type (
	// Resolves extends Provides semantics to dispatch a
	// Callback on the resolved handler.
	Resolves struct {
		Provides
		greedy    bool
		succeeded bool
	}

	// ResolvesBuilder builds Resolves callbacks.
	ResolvesBuilder struct {
		ProvidesBuilder
		greedy   bool
	}
)

// Resolves

func (r *Resolves) Succeeded() bool {
	return r.succeeded
}

func (r *Resolves) accept(
	result   any,
	composer Handler,
) HandleResult {
	if greedy := r.greedy; !greedy && r.succeeded {
		return Handled
	} else {
		hr := DispatchCallback(result, r.Trigger(), greedy, composer)
		r.succeeded = r.succeeded || hr.handled
		return hr
	}
}

// ResolvesBuilder

func (b *ResolvesBuilder) WithGreedy(
	greedy bool,
) *ResolvesBuilder {
	b.greedy = greedy
	return b
}

func (b *ResolvesBuilder) New(callback Callback) *Resolves {
	resolves := &Resolves{
		Provides: b.WithTrigger(callback).Build(),
		greedy:   b.greedy,
	}
	resolves.SetAcceptResult(resolves.accept)
	return resolves
}
