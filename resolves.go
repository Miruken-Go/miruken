package miruken

type resolves struct {
	Provides
	callback  Callback
	greedy    bool
	succeeded bool
}

func (r *resolves) Callback() Callback {
	return r.callback
}

func (r *resolves) Succeeded() bool {
	return r.succeeded
}

func (r *resolves) CanDispatch(
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

func (r *resolves) accept(
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

type resolvesBuilder struct {
	ProvidesBuilder
	callback Callback
	greedy   bool
}

func (b *resolvesBuilder) WithCallback(
	callback Callback,
) *resolvesBuilder {
	b.callback = callback
	return b
}

func (b *resolvesBuilder) WithGreedy(
	greedy bool,
) *resolvesBuilder {
	b.greedy = greedy
	return b
}

func (b *resolvesBuilder) New() *resolves {
	resolves := &resolves{
		Provides: b.Build(),
		callback: b.callback,
		greedy:   b.greedy,
	}
	resolves.SetAcceptResult(resolves.accept)
	return resolves
}
