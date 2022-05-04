package miruken

type Resolving struct {
	Provides
	callback  Callback
	greedy    bool
	succeeded bool
}

func (r *Resolving) Succeeded() bool {
	return r.succeeded
}

func (r *Resolving) CanDispatch(
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

func (r *Resolving) accept(
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

type ResolvingBuilder struct {
	ProvidesBuilder
	callback Callback
	greedy   bool
}

func (b *ResolvingBuilder) WithCallback(
	callback Callback,
) *ResolvingBuilder {
	b.callback = callback
	return b
}

func (b *ResolvingBuilder) WithGreedy(
	greedy bool,
) *ResolvingBuilder {
	b.greedy = greedy
	return b
}

func (b *ResolvingBuilder) NewResolving() *Resolving {
	b.WithMany()
	resolving := &Resolving{
		Provides: b.Provides(),
		callback: b.callback,
		greedy:   b.greedy,
	}
	resolving.CallbackBase.accept = resolving.accept
	return resolving
}
