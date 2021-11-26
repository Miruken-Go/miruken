package miruken

import (
	"reflect"
)

// Provides results Covariantly.
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
	count := len(p.results)
	return DispatchPolicy(handler, p, p, greedy, composer, p).
		OtherwiseHandledIf(len(p.results) > count)
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
	if resolution == nil {
		return false
	}
	if strict {
		if included = p.AcceptResult(resolution, greedy, composer); included {
			p.results = append(p.results, resolution)
		}
		return included
	}
	switch reflect.TypeOf(resolution).Kind() {
	case reflect.Slice, reflect.Array:
		forEach(resolution, func(idx int, value interface{}) {
			if value != nil {
				if inc := p.AcceptResult(value, greedy, composer); inc {
					p.results = append(p.results, value)
					included  = true
				}
			}
		})
	default:
		if included = p.AcceptResult(resolution, greedy, composer); included {
			p.results = append(p.results, resolution)
		}
	}
	return included
}

type InquiryBuilder struct {
	CallbackBuilder
	key          interface{}
	parent      *Provides
	constraints  []func(*ConstraintBuilder)
}

func (b *InquiryBuilder) WithKey(
	key interface{},
) *InquiryBuilder {
	b.key = key
	return b
}

func (b *InquiryBuilder) WithParent(
	parent *Provides,
) *InquiryBuilder {
	b.parent = parent
	return b
}

func (b *InquiryBuilder) WithConstraints(
	constraints ... func(*ConstraintBuilder),
) *InquiryBuilder {
	if len(constraints) > 0 {
		b.constraints = append(b.constraints, constraints...)
	}
	return b
}

func (b *InquiryBuilder) Inquiry() Provides {
	inquiry := Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
	}
	ApplyConstraints(&inquiry, b.constraints...)
	return inquiry
}

func (b *InquiryBuilder) NewInquiry() *Provides {
	inquiry := &Provides{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		parent:       b.parent,
	}
	ApplyConstraints(inquiry, b.constraints...)
	return inquiry
}

func Resolve(
	handler     Handler,
	target      interface{},
	constraints ... func(*ConstraintBuilder),
) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv       := TargetValue(target)
	inquiry  := new(InquiryBuilder).
		WithKey(tv.Type().Elem()).
		WithConstraints(constraints...).
		NewInquiry()
	if result := handler.Handle(inquiry, false, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
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
	tv      := TargetSliceValue(target)
	builder := new(InquiryBuilder).
		WithKey(tv.Type().Elem().Elem()).
		WithConstraints(constraints...)
	builder.WithMany()
	inquiry := builder.NewInquiry()
	if result := handler.Handle(inquiry, true, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
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