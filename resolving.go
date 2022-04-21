package miruken

type Resolving struct {
	Provides
	callback  Callback
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
	greedy   bool,
	composer Handler,
) bool {
	if many := r.callback.Many(); !many && r.succeeded {
		return true
	} else if DispatchCallback(result, r.callback, many, composer).handled {
		r.succeeded = true
		return true
	}
	return false
}

type ResolvingBuilder struct {
	ProvidesBuilder
	callback Callback
}

func (b *ResolvingBuilder) WithCallback(
	callback Callback,
) *ResolvingBuilder {
	b.callback = callback
	return b
}

func (b *ResolvingBuilder) NewResolving() *Resolving {
	b.WithMany()
	resolving := &Resolving{
		Provides: b.Provides(),
		callback: b.callback,
	}
	resolving.CallbackBase.accept = resolving.accept
	return resolving
}
