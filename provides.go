package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// Provides results covariantly.
type Provides struct {
	CallbackBase
	key     any
	parent  *Provides
	handler any
	binding Binding
	owner   any
}

func (p *Provides) Key() any {
	return p.key
}

func (p *Provides) Policy() Policy {
	return providesPolicyIns
}

func (p *Provides) Parent() *Provides {
	return p.parent
}

func (p *Provides) Binding() Binding {
	return p.binding
}

func (p *Provides) Owner() any {
	return p.owner
}

func (p *Provides) CanDispatch(
	handler any,
	binding Binding,
) (reset func (), approved bool) {
	if p.inProgress(handler, binding) {
		return nil, false
	}
	return func(h any, b Binding) func () {
		p.handler, p.binding = handler, binding
		return func () {
			p.handler, p.binding = h, b
		}
	}(p.handler, p.binding), true
}

func (p *Provides) inProgress(
	handler any,
	binding Binding,
) bool {
	if p.handler == handler && p.binding == binding {
		return true
	}
	if parent := p.parent; parent != nil {
		return parent.inProgress(handler, binding)
	}
	return false
}

func (p *Provides) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) (result HandleResult) {
	result = NotHandled
	count := p.ResultCount()
	if len(p.Constraints()) == 0 {
		if typ, ok := p.key.(reflect.Type); ok {
			if reflect.TypeOf(handler).AssignableTo(typ) {
				result = result.Or(p.ReceiveResult(handler, false, composer))
				if result.stop || (result.handled && !greedy) {
					return result
				}
			}
		}
	}
	return DispatchPolicy(handler, p, greedy, composer).
		OtherwiseHandledIf(p.ResultCount() > count)
}

func (p *Provides) String() string {
	return fmt.Sprintf("provides %+v", p.key)
}

func (p *Provides) Resolve(
	handler Handler,
	many    bool,
) (any, *promise.Promise[any], error) {
	if result := handler.Handle(p, many, nil); result.IsError() {
		return nil, nil, result.Error()
	}
	r, pr := p.Result(many)
	return r, pr, nil
}

func (p *Provides) acceptPromise(
	pa *promise.Promise[any],
) *promise.Promise[any] {
	return promise.Catch(pa, func(error) error {
		return nil
	})
}

// ProvidesBuilder builds Provides callbacks.
type ProvidesBuilder struct {
	CallbackBuilder
	key      any
	parent  *Provides
	owner    any
}

func (b *ProvidesBuilder) WithKey(
	key any,
) *ProvidesBuilder {
	if internal.IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
}

func (b *ProvidesBuilder) WithParent(
	parent *Provides,
) *ProvidesBuilder {
	b.parent = parent
	return b
}

func (b *ProvidesBuilder) ForOwner(
	owner any,
) *ProvidesBuilder {
	b.owner = owner
	return b
}

func (b *ProvidesBuilder) Build() Provides {
	return Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
	}
}

func (b *ProvidesBuilder) New() *Provides {
	p := &Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
		owner:        b.owner,
	}
	p.SetAcceptPromiseResult(p.acceptPromise)
	return p
}

func Resolve[T any](
	handler     Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return ResolveKey[T](handler, internal.TypeOf[T](), constraints...)
}

func ResolveKey[T any](
	handler     Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder ProvidesBuilder
	builder.WithKey(key).
		    IntoTarget(&t).
		    WithConstraints(constraints...)
	provides := builder.New()
	if result := handler.Handle(provides, false, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		if _, p := provides.Result(false); p != nil {
			tp = promise.Then(p, func(any) T {
				return t
			})
		}
	}
	return
}

func ResolveAll[T any](
	handler     Handler,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder ProvidesBuilder
	builder.WithKey(internal.TypeOf[T]()).
		    IntoTarget(&t).
		    WithConstraints(constraints...)
	provides := builder.New()
	if result := handler.Handle(provides, true, nil); result.IsError() {
		err = result.Error()
	} else if result.handled {
		if _, p := provides.Result(true); p != nil {
			tp = promise.Then(p, func(any) []T {
				return t
			})
		}
	}
	return
}


// providesPolicy for providing instances covariantly with lifestyle.
type providesPolicy struct {
	CovariantPolicy
}

func (p *providesPolicy) NewCtorBinding(
	handlerType reflect.Type,
	constructor *reflect.Method,
	spec        *bindingSpec,
	key         any,
) (binding Binding, err error) {
	explicitSpec := spec != nil
	if !explicitSpec {
		single := &Single{}
		if err = single.Init(); err != nil {
			return nil, err
		}
		spec = &bindingSpec{filters: []FilterProvider{single}}
	}
	return newCtorBinding(handlerType, constructor, spec, key, explicitSpec)
}

var providesPolicyIns Policy = &providesPolicy{}