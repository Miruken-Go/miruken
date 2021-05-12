package callback

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	_policyType = reflect.TypeOf((*Policy)(nil)).Elem()
)

type key struct{}

// HandlerDescriptor

type HandlerDescriptor struct {
	owner    reflect.Type
	bindings map[Policy][]Binding
}

func (d *HandlerDescriptor) Dispatch(
	policy   Policy,
	handler  interface{},
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
	results  ResultReceiver,
) HandleResult {
	return NotHandled
}

// HandlerDescriptorError

type HandlerDescriptorError struct {
	HandlerType reflect.Type
	Reason      error
}

func (e *HandlerDescriptorError) Error() string {
	return fmt.Sprintf("handler type %d: reason %v", e.HandlerType, e.Reason)
}

func (e *HandlerDescriptorError) Unwrap() error { return e.Reason }

// HandlerDescriptorFactory

type HandlerDescriptorProvider interface {
	GetHandlerDescriptor(handler reflect.Type) *HandlerDescriptor
}

// HandlerDescriptorFactory

type HandlerDescriptorFactory interface {
	HandlerDescriptorProvider
	RegisterHandlerType(handlerType reflect.Type) (*HandlerDescriptor, error)
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
	binding     Binding,
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
) *HandlerDescriptor {
	if handlerType == nil {
		panic("nil handlerType")
	}
	f.RLock()
	defer f.RUnlock()
	return f.descriptors[handlerType]
}

func (f *mutableFactory) RegisterHandlerType(
	handlerType reflect.Type,
) (*HandlerDescriptor, error) {
	if handlerType == nil {
		panic("nil handlerType")
	}

	f.Lock()
	defer f.Unlock()

	if descriptor := f.descriptors[handlerType]; descriptor != nil {
		return descriptor, nil
	}

	bindings := make(map[Policy][]Binding)

	for i := 0; i < handlerType.NumMethod(); i++ {
		method     := handlerType.Method(i)
		methodType := method.Type

		if methodType.NumIn() < 2 || methodType.IsVariadic() {
			continue
		}

		policyType := methodType.In(1)

		if !reflect.PtrTo(policyType).Implements(_policyType) {
			continue
		}

		policy := requirePolicy(policyType)
		if binding := policy.BindingFor(method); binding != nil {
			policyBindings, found := bindings[policy]
			policyBindings = append(policyBindings, binding)
			if !found {
				bindings[policy] = policyBindings
			}
		}
	}

	return nil, nil
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
	opts ... MutableHandlerDescriptorFactoryOption,
) HandlerDescriptorFactory {
	factory := &mutableFactory{}

	for _, opt := range opts {
		opt.applyMutableFactoryOption(factory)
	}

	return factory
}
