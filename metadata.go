package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"sync"
)

type key struct{}

// HandlerDescriptor

type HandlerDescriptor struct {
	handlerType reflect.Type
	bindings    map[Policy][]Binding
}

func (d *HandlerDescriptor) Dispatch(
	policy      Policy,
	handler     interface{},
	callback    interface{},
	rawCallback interface{},
	greedy      bool,
	ctx         HandleContext,
	results     ResultReceiver,
) (result HandleResult) {
	result = NotHandled
	if policyBindings, found := d.bindings[policy]; found {
		constraint := policy.Constraint(callback)
		for _, binding := range policyBindings {
			if result.stop || (result.handled && !greedy) {
				return result
			}
			if binding.Matches(constraint, policy.Variance()) {
				output   := binding.Invoke(handler, callback, rawCallback, ctx)
				res, accepted := policy.AcceptResults(output)
				if accepted.IsHandled() && results != nil &&
					results.ReceiveResult(res, false, greedy, ctx) {
					accepted = accepted.Or(Handled)
				}
				result = result.Or(accepted)
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

	if descriptor, err = newHandlerDescriptor(handlerType); err == nil {
		f.descriptors[handlerType] = descriptor
	}

	return descriptor, err
}

func newHandlerDescriptor(
	handlerType reflect.Type,
) (descriptor *HandlerDescriptor, invalid error) {
	bindings := make(map[Policy][]Binding)

	for i := 0; i < handlerType.NumMethod(); i++ {
		method     := handlerType.Method(i)
		methodType := method.Type

		if methodType.NumIn() < 2 {
			continue
		}

		policyType := methodType.In(1)
		if !isPolicy(policyType) {
			continue
		}
		policy := requirePolicy(policyType)
		if binder, ok := policy.(MethodBinder);ok {
			if binding, err := binder.NewMethodBinding(method); binding != nil {
				policyBindings, _ := bindings[policy]
				policyBindings   = append(policyBindings, binding)
				bindings[policy] = policyBindings
			} else if err != nil {
				invalid = multierror.Append(invalid, err)
			}
		}
	}

	if invalid != nil {
		return nil, &HandlerDescriptorError{handlerType, invalid}
	}

	return &HandlerDescriptor{handlerType, bindings}, nil
}

func validHandlerType(handlerType reflect.Type) error {
	if handlerType == nil {
		panic("nil handlerType")
	}
	t := handlerType
	if kind := t.Kind(); kind == reflect.Ptr {
		t = t.Elem()
	}
	if kind := t.Kind(); kind != reflect.Struct {
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


func WithVisitor(
	visitor HandlerDescriptorVisitor,
) MutableHandlerDescriptorFactoryOption {
	return mutableFactoryOptionFunc(func (factory *mutableFactory) {
		factory.visitor = visitor
	})
}

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
