package miruken

import (
	"container/list"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"sync"
)

// PolicyBindings

type PolicyBindings struct {
	order     OrderBinding
	bindings *list.List
}

func (p *PolicyBindings) insert(binding Binding) {
	p.bindings.PushBack(binding)
}

func newPolicyBindings(order OrderBinding) *PolicyBindings {
	if order == nil {
		panic("order cannot be nil")
	}
	return &PolicyBindings{order, list.New()}
}

// HandlerDescriptor

type HandlerDescriptor struct {
	handlerType reflect.Type
	bindings    map[Policy]*PolicyBindings
}

func (d *HandlerDescriptor) Dispatch(
	policy      Policy,
	handler     interface{},
	callback    interface{},
	rawCallback interface{},
	constraint  interface{},
	greedy      bool,
	composer    Handler,
	results     ResultReceiver,
) (result HandleResult) {
	result = NotHandled
	if pb, found := d.bindings[policy]; found {
		if constraint == nil {
			switch typ := callback.(type) {
			case reflect.Type: constraint = typ
			default: constraint = reflect.TypeOf(callback)
			}
		}
		for e := pb.bindings.Front(); e != nil; e = e.Next() {
			if result.stop || (result.handled && !greedy) {
				return result
			}
			binding := e.Value.(Binding)
			if binding.Matches(constraint, policy.Variance()) {
				var approved bool
				var reset func (rawCallback interface{})
				if guard, ok := rawCallback.(CallbackGuard); ok {
					if reset, approved = guard.CanDispatch(handler, binding); !approved {
						continue
					}
				}
				if out, err := binding.Invoke(handler, callback, rawCallback, composer); err == nil {
					res, accepted := policy.AcceptResults(out)
					if accepted.IsHandled() && results != nil &&
						results.ReceiveResult(res, binding.Strict(), greedy, composer) {
						accepted = accepted.Or(Handled)
					}
					result = result.Or(accepted)
				}
				if reset != nil {
					reset(rawCallback)
				}
			}
		}
	}
	return result
}

// HandlerDescriptorError

type HandlerDescriptorError struct {
	HandlerType reflect.Type
	Reason      error
}

func (e *HandlerDescriptorError) Error() string {
	return fmt.Sprintf("invalid handler: %v reason: %v", e.HandlerType, e.Reason)
}

func (e *HandlerDescriptorError) Unwrap() error { return e.Reason }

// HandlerDescriptorFactory

type HandlerDescriptorProvider interface {
	GetHandlerDescriptor(
		handler reflect.Type,
	) (*HandlerDescriptor, error)
}

// HandlerDescriptorFactory

type HandlerDescriptorFactory interface {
	HandlerDescriptorProvider

	RegisterHandlerType(
		handlerType reflect.Type,
	) (*HandlerDescriptor, error)
}

type HandlerDescriptorVisitor interface {
	VisitHandlerBinding(
		descriptor *HandlerDescriptor,
		binding     Binding,
	)
}

type HandlerDescriptorVisitorFunc func(*HandlerDescriptor, Binding)

func (f HandlerDescriptorVisitorFunc) VisitHandlerBinding(
	descriptor *HandlerDescriptor,
	binding Binding,
) {
	f(descriptor, binding)
}

// mutableHandlerDescriptorFactory

type mutableFactory struct {
	sync.RWMutex
	descriptors map[reflect.Type]*HandlerDescriptor
	visitor     HandlerDescriptorVisitor
}

func (f *mutableFactory) GetHandlerDescriptor(
	handlerType reflect.Type,
) (descriptor *HandlerDescriptor, err error) {
	if err = validHandlerType(handlerType); err != nil {
		return nil, err
	}
	f.RLock()
	defer f.RUnlock()
	return f.descriptors[handlerType], nil
}

func (f *mutableFactory) RegisterHandlerType(
	handlerType reflect.Type,
) (descriptor *HandlerDescriptor, err error) {
	if err = validHandlerType(handlerType); err != nil {
		return nil, err
	}

	f.Lock()
	defer f.Unlock()

	if descriptor := f.descriptors[handlerType]; descriptor != nil {
		return descriptor, nil
	}
	if descriptor, err = f.newHandlerDescriptor(handlerType); err == nil {
		f.descriptors[handlerType] = descriptor
	}
	return descriptor, err
}

func (f *mutableFactory) newHandlerDescriptor(
	handlerType reflect.Type,
) (descriptor *HandlerDescriptor, invalid error) {
	descriptor = &HandlerDescriptor{
		handlerType: handlerType,
	}
	bindings := make(map[Policy]*PolicyBindings)
	for i := 0; i < handlerType.NumMethod(); i++ {
		method     := handlerType.Method(i)
		methodType := method.Type
		if methodType.NumIn() < 2 {
			continue
		}
		if spec, err := buildPolicySpec(methodType.In(1)); err == nil {
			if spec == nil {
				continue
			}
			policy := spec.policy
			if binder, ok := policy.(methodBinder); ok {
				if binding, errBind := binder.newMethodBinding(method, spec); binding != nil {
					policyBindings, found := bindings[policy]
					if !found {
						policyBindings = newPolicyBindings(policy)
						bindings[policy] = policyBindings
					}
					if f.visitor != nil {
						f.visitor.VisitHandlerBinding(descriptor, binding)
					}
					policyBindings.insert(binding)
				} else if errBind != nil {
					invalid = multierror.Append(invalid, errBind)
				}
			}
		} else {
			invalid = multierror.Append(invalid, err)
		}
	}
	if invalid != nil {
		return nil, &HandlerDescriptorError{handlerType, invalid}
	}
	descriptor.bindings = bindings
	return descriptor, nil
}

func validHandlerType(handlerType reflect.Type) error {
	if handlerType == nil {
		panic("handlerType cannot be nil")
	}
	typ := handlerType
	if kind := typ.Kind(); kind == reflect.Ptr {
		typ = typ.Elem()
	}
	if kind := typ.Kind(); kind != reflect.Struct {
		return fmt.Errorf("handler: %v is not a struct or *struct", handlerType)
	}
	return nil
}

type MutableHandlerDescriptorFactoryOption interface {
	applyMutableFactoryOption(factory *mutableFactory)
}

type mutableFactoryOptionFunc func(*mutableFactory)

func (f mutableFactoryOptionFunc) applyMutableFactoryOption(
	factory *mutableFactory,
) { f(factory) }

func NewMutableHandlerDescriptorFactory(
	opts ...MutableHandlerDescriptorFactoryOption,
) HandlerDescriptorFactory {
	factory := &mutableFactory{
		descriptors: make(map[reflect.Type]*HandlerDescriptor),
	}
	for _, opt := range opts {
		opt.applyMutableFactoryOption(factory)
	}
	return factory
}

func WithHandlerDescriptorVisitor(
	visitor HandlerDescriptorVisitor,
) MutableHandlerDescriptorFactoryOption {
	return mutableFactoryOptionFunc(func (factory *mutableFactory) {
		factory.visitor = visitor
	})
}
