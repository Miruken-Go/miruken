package miruken

import "reflect"

type Inquiry struct {
	CallbackBase
	key      interface{}
	parent  *Inquiry
	handler  interface{}
	binding  Binding
}

func (i *Inquiry) Key() interface{} {
	return i.key
}

func (i *Inquiry) Parent() *Inquiry {
	return i.parent
}

func (i *Inquiry) Policy() Policy {
	return ProvidesPolicy()
}

func (i *Inquiry) ReceiveResult(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return i.include(result, strict, greedy, composer)
}

func (i *Inquiry) CanDispatch(
	handler interface{},
	binding Binding,
) (reset func (), approved bool) {
	if i.inProgress(handler, binding) {
		return nil, false
	}
	return func(h interface{}, b Binding) func () {
		return func () {
			i.handler = h
			i.binding = b
		}
	}(i.handler, i.binding), true
}

func (i *Inquiry)inProgress(
	handler interface{},
	binding Binding,
) bool {
	if i.handler == handler && i.binding == binding {
		return true
	}
	if parent := i.parent; parent != nil {
		return parent.inProgress(handler, binding)
	}
	return false
}

func (i *Inquiry) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) (result HandleResult) {
	result = NotHandled
	if typ, ok := i.key.(reflect.Type); ok {
		if reflect.TypeOf(handler).AssignableTo(typ) {
			resolved := i.ReceiveResult(handler, false, greedy, composer)
			result = result.OtherwiseHandledIf(resolved)
			if resolved && !greedy {
				return result
			}
		}
	}
	count := len(i.results)
	return DispatchPolicy(i.Policy(), handler, i, i, i.key, greedy, composer, i).
		OtherwiseHandledIf(len(i.results) > count)
}

func (i *Inquiry) Resolve(
	handler Handler,
) (interface{}, error) {
	if result := handler.Handle(i, i.Many(), nil); result.IsError() {
		return nil, result.Error()
	}
	return i.Result(), nil
}

func (i *Inquiry) include(
	resolution interface{},
	strict     bool,
	greedy     bool,
	composer   Handler,
) (included bool) {
	if resolution == nil {
		return false
	}
	if strict {
		if included = i.AcceptResult(resolution, greedy, composer); included {
			i.results = append(i.results, resolution)
		}
		return included
	}
	switch reflect.TypeOf(resolution).Kind() {
	case reflect.Slice, reflect.Array:
		forEach(resolution, func(idx int, value interface{}) {
			if value != nil {
				if inc := i.AcceptResult(value, greedy, composer); inc {
					i.results = append(i.results, value)
					included  = true
				}
			}
		})
	default:
		if included = i.AcceptResult(resolution, greedy, composer); included {
			i.results = append(i.results, resolution)
		}
	}
	return included
}

type InquiryBuilder struct {
	CallbackBuilder
	key      interface{}
	parent  *Inquiry
}

func (b *InquiryBuilder) WithKey(
	key interface{},
) *InquiryBuilder {
	b.key = key
	return b
}

func (b *InquiryBuilder) WithParent(
	parent *Inquiry,
) *InquiryBuilder {
	b.parent = parent
	return b
}

func (b *InquiryBuilder) Inquiry() Inquiry {
	return Inquiry{
		CallbackBase: b.Callback(),
		key:          b.key,
		parent:       b.parent,
	}
}

func (b *InquiryBuilder) NewInquiry() *Inquiry {
	return &Inquiry{
		CallbackBase: b.Callback(),
		key:          b.key,
		parent:       b.parent,
	}
}

func Resolve(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv       := TargetValue(target)
	inquiry  := new(InquiryBuilder).
		WithKey(tv.Type().Elem()).
		NewInquiry()
	if result := handler.Handle(inquiry, false, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
	return nil
}

func ResolveAll(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	builder := new(InquiryBuilder).
		WithKey(tv.Type().Elem().Elem())
	builder.WithMany()
	inquiry := builder.NewInquiry()
	if result := handler.Handle(inquiry, true, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
	return nil
}