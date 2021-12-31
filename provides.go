package miruken

import (
	"reflect"
)

// Provides results covariantly.
type Provides struct {
	CallbackBase
	key       interface{}
	parent   *Provides
	handler   interface{}
	binding   Binding
	metadata  BindingMetadata
}

func (p *Provides) Key() interface{} {
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
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return p.include(result, strict, greedy, composer)
}

func (p *Provides) CanDispatch(
	handler interface{},
	binding Binding,
) (reset func (), approved bool) {
	if p.inProgress(handler, binding) {
		return nil, false
	}
	return func(h interface{}, b Binding) func () {
		p.handler = handler
		p.binding = binding
		return func () {
			p.handler = h
			p.binding = b
		}
	}(p.handler, p.binding), true
}

func (p *Provides)inProgress(
	handler interface{},
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
	handler  interface{},
	greedy   bool,
	composer Handler,
) (result HandleResult) {
	result = NotHandled
	if p.metadata.IsEmpty() {
		if typ, ok := p.key.(reflect.Type); ok {
			if reflect.TypeOf(handler).AssignableTo(typ) {
				resolved := p.ReceiveResult(handler, false, greedy, composer)
				result = result.OtherwiseHandledIf(resolved)
				if resolved && !greedy {
					return result
				}
			}
		}
	}
	return DispatchPolicy(handler, p, p, greedy, composer)
}

func (p *Provides) Resolve(
	handler Handler,
) (interface{}, error) {
	if result := handler.Handle(p, p.Many(), nil); result.IsError() {
		return nil, result.Error()
	}
	return p.Result(), nil
}

func (p *Provides) include(
	resolution interface{},
	strict     bool,
	greedy     bool,
	composer   Handler,
) (included bool) {
	if IsNil(resolution) {
		return false
	}
	if strict {
		return p.AddResult(resolution, greedy, composer)
	}
	switch reflect.TypeOf(resolution).Kind() {
	case reflect.Slice, reflect.Array:
		forEach(resolution, func(idx int, value interface{}) {
			if value != nil {
				included = p.AddResult(value, greedy, composer) || included
			}
		})
	default:
		included = p.AddResult(resolution, greedy, composer)
	}
	return included
}

// ProvidesBuilder builds Provides callbacks.
type ProvidesBuilder struct {
	CallbackBuilder
	key          interface{}
	parent      *Provides
	constraints  []func(*ConstraintBuilder)
}

func (b *ProvidesBuilder) WithKey(
	key interface{},
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

func Resolve(
	handler     Handler,
	target      interface{},
	constraints ... func(*ConstraintBuilder),
) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv := TargetValue(target)
	var builder ProvidesBuilder
	provides := builder.
		WithKey(tv.Type().Elem()).
		WithConstraints(constraints...).
		NewProvides()
	if result := handler.Handle(provides, false, nil); result.IsError() {
		return result.Error()
	}
	provides.CopyResult(tv)
	return nil
}

func ResolveAll(
	handler     Handler,
	target      interface{},
	constraints ... func(*ConstraintBuilder),
) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv := TargetSliceValue(target)
	var builder ProvidesBuilder
	builder.WithKey(tv.Type().Elem().Elem()).
		    WithConstraints(constraints...)
	builder.WithMany()
	provides := builder.NewProvides()
	if result := handler.Handle(provides, true, nil); result.IsError() {
		return result.Error()
	}
	provides.CopyResult(tv)
	return nil
}

// providesPolicy for providing instances with lifestyles covariantly.
type providesPolicy struct {
	CovariantPolicy
}

func (p *providesPolicy) NewConstructorBinding(
	handlerType  reflect.Type,
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