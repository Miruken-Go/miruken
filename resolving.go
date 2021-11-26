package miruken

type Resolving struct {
	Provides
	callback  interface{}
	succeeded bool
}

func (r *Resolving) Succeeded() bool {
	return r.succeeded
}

func (r *Resolving) CanDispatch(
	handler interface{},
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
	result   interface{},
	greedy   bool,
	composer Handler,
) bool {
	if r.succeeded && !greedy {
		return true
	}
	if DispatchCallback(result, r.callback, greedy, composer).handled {
		r.succeeded = true
		return true
	}
	return false
}

type ResolvingBuilder struct {
	ProvidesBuilder
	callback interface{}
}

func (b *ResolvingBuilder) WithCallback(
	callback interface{},
) *ResolvingBuilder {
	b.callback = callback
	return b
}

func (b *ResolvingBuilder) NewResolving() *Resolving {
	resolving := &Resolving{
		Provides: b.Provides(),
		callback: b.callback,
	}
	resolving.CallbackBase.accept = AcceptResultFunc(resolving.accept)
	return resolving
}
