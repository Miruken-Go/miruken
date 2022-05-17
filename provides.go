package miruken

import (
	"reflect"
)

// Provides results covariantly.
type Provides struct {
	CallbackBase
	key       any
	parent   *Provides
	handler   any
	binding   Binding
	metadata  BindingMetadata
}

func (p *Provides) Key() any {
	return p.key
}

func (p *Provides) Policy() Policy {
	return _providesPolicy
}

func (p *Provides) Parent() *Provides {
	return p.parent
}

func (p *Provides) Binding() Binding {
	return p.binding
}

func (p *Provides) Metadata() *BindingMetadata {
	return &p.metadata
}

func (p *Provides) ReceiveResult(
	result   any,
	strict   bool,
	composer Handler,
) HandleResult {
	return p.include(result, strict, composer)
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

func (p *Provides)inProgress(
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
	if p.metadata.IsEmpty() {
		if typ, ok := p.key.(reflect.Type); ok {
			if reflect.TypeOf(handler).AssignableTo(typ) {
				result = result.Or(p.ReceiveResult(handler, false, composer))
				if result.stop || (result.handled && !greedy) {
					return result
				}
			}
		}
	}
	return DispatchPolicy(handler, p, greedy, composer)
}

func (p *Provides) Resolve(
	handler Handler,
	many    bool,
) (any, error) {
	if result := handler.Handle(p, many, nil); result.IsError() {
		return nil, result.Error()
	}
	return p.Result(many), nil
}

func (p *Provides) include(
	resolution any,
	strict     bool,
	composer   Handler,
) (result HandleResult) {
	if IsNil(resolution) {
		return NotHandled
	}
	if strict {
		return p.AddResult(resolution, composer)
	}
	result = NotHandled
	switch reflect.TypeOf(resolution).Kind() {
	case reflect.Slice, reflect.Array:
		forEach(resolution, func(idx int, value any) bool {
			if value != nil {
				result = result.Or(p.AddResult(value, composer))
				return result.stop
			}
			return false
		})
	default:
		return p.AddResult(resolution, composer)
	}
	return result
}

// ProvidesBuilder builds Provides callbacks.
type ProvidesBuilder struct {
	CallbackBuilder
	key          any
	parent      *Provides
	constraints  []func(*ConstraintBuilder)
}

func (b *ProvidesBuilder) WithKey(
	key any,
) *ProvidesBuilder {
	if IsNil(key) {
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

func (b *ProvidesBuilder) WithConstraints(
	constraints ... func(*ConstraintBuilder),
) *ProvidesBuilder {
	if len(constraints) > 0 {
		b.constraints = append(b.constraints, constraints...)
	}
	return b
}

func (b *ProvidesBuilder) Provides() Provides {
	provides := Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
	}
	ApplyConstraints(&provides, b.constraints...)
	return provides
}

func (b *ProvidesBuilder) NewProvides() *Provides {
	provides := &Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
	}
	ApplyConstraints(provides, b.constraints...)
	return provides
}

func Resolve[T any](
	handler     Handler,
	constraints ... func(*ConstraintBuilder),
) (T, error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	var target T
	tv := TargetValue(&target)
	var builder ProvidesBuilder
	provides := builder.
		WithKey(tv.Type().Elem()).
		WithConstraints(constraints...).
		NewProvides()
	if result := handler.Handle(provides, false, nil); result.IsError() {
		return target, result.Error()
	}
	provides.CopyResult(tv, false)
	return target, nil
}

func ResolveAll[T any](
	handler     Handler,
	constraints ... func(*ConstraintBuilder),
) ([]T, error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	var target []T
	tv := TargetSliceValue(&target)
	var builder ProvidesBuilder
	builder.WithKey(tv.Type().Elem().Elem()).
		    WithConstraints(constraints...)
	provides := builder.NewProvides()
	if result := handler.Handle(provides, true, nil); result.IsError() {
		return target, result.Error()
	}
	provides.CopyResult(tv, true)
	return target, nil
}

// providesPolicy for providing instances covariantly with lifestyle.
type providesPolicy struct {
	CovariantPolicy
}

func (p *providesPolicy) NewConstructorBinding(
	handlerType reflect.Type,
	constructor *reflect.Method,
	spec        *policySpec,
) (binding Binding, err error) {
	explicitSpec := spec != nil
	if !explicitSpec {
		single := new(Singleton)
		if err = single.Init(); err != nil {
			return nil, err
		}
		spec = &policySpec{
			filters: []FilterProvider{single},
		}
	}
	return newConstructorBinding(handlerType, constructor, spec, explicitSpec)
}

var _providesPolicy Policy = &providesPolicy{}