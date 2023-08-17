package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// Provides results covariantly.
	Provides struct {
		CallbackBase
		key     any
		parent  *Provides
		handler any
		binding Binding
		owner   any
	}

	// ProvidesBuilder builds Provides callbacks.
 	ProvidesBuilder struct {
		CallbackBuilder
		key      any
		parent  *Provides
		owner    any
	}

	// providesPolicy provides values covariantly with lifestyle.
 	providesPolicy struct {
		CovariantPolicy
	}
)


// Provides

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


// ProvidesBuilder

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


// Resolve retrieves a value of type parameter T.
// Applies any lifestyle if present.
func Resolve[T any](
	handler     Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return ResolveKey[T](handler, internal.TypeOf[T](), constraints...)
}

// ResolveKey retrieves a value of type parameter T with the specified key.
// Applies any lifestyle if present.
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

// ResolveAll retrieves all values of type parameter T.
// Applies any lifestyle if present.
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


// providesPolicy

func (p *providesPolicy) NewCtorBinding(
	typ   reflect.Type,
	ctor  *reflect.Method,
	inits []reflect.Method,
	spec  *bindingSpec,
	key   any,
) (Binding, error) {
	binding, err := p.CovariantPolicy.NewCtorBinding(typ, ctor, inits, spec, key)
	if err == nil {
		if spec == nil {
			binding.AddFilters(&Single{})
		}
		if err = initLifestyles(binding); err != nil {
			return nil, err
		}
	}
	return binding, err
}

func (p *providesPolicy) NewMethodBinding(
	method reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	binding, err := p.CovariantPolicy.NewMethodBinding(method, spec, key)
	if err == nil {
		if err = initLifestyles(binding); err != nil {
			return nil, err
		}
	}
	return binding, err
}

func (p *providesPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	binding, err := p.CovariantPolicy.NewFuncBinding(fun, spec, key)
	if err == nil {
		if err = initLifestyles(binding); err != nil {
			return nil, err
		}
	}
	return binding, err
}


func initLifestyles(binding Binding) error {
	for _, filter := range binding.Filters() {
		if lifestyle, ok := filter.(LifestyleInit); ok {
			if err := lifestyle.InitLifestyle(binding); err != nil {
				return err
			}
		}
	}
	return nil
}


var providesPolicyIns Policy = &providesPolicy{}