package callback

import (
	"reflect"
	"sync"
)

type key struct{}

// Binding

type Binding struct {

}

// HandlerDescriptor

type HandlerDescriptor struct {
	owner reflect.Type
}

func (d *HandlerDescriptor) Dispatch(
	policy   Policy,
	callback interface{},
	greedy   bool,
	context  HandleContext,
	results  ResultReceiver,
) HandleResult {
	return NotHandled
}

// HandlerDescriptorFactory

type HandlerDescriptorFactory interface {
	GetHandlerDescriptor(handler interface{}) *HandlerDescriptor
	RegisterHandlerDescriptor(handlerType reflect.Type) (*HandlerDescriptor, error)
}

// mutableHandlerDescriptorFactory

type mutableFactory struct {
	sync.RWMutex
	descriptors map[reflect.Type]*HandlerDescriptor
}

func (f *mutableFactory) GetHandlerDescriptor(
	handler interface{},
) *HandlerDescriptor {
	if handler == nil {
		panic("nil handler")
	}
	f.RLock()

	defer f.RUnlock()
	return f.descriptors[reflect.TypeOf(handler)]
}

func (f *mutableFactory) RegisterHandlerDescriptor(
	handlerType reflect.Type,
) (*HandlerDescriptor, error) {
	f.Lock()
	defer f.Unlock()
	return nil, nil
}

func NewMutableHandlerDescriptorFactory() HandlerDescriptorFactory {
	return &mutableFactory{}
}

var factoryKey key

func WithHandlerDescriptorFactory(
	parent  HandleContext,
	factory HandlerDescriptorFactory,
) HandleContext {
	if factory == nil {
		panic("nil factory")
	}
	return WithKeyValue(parent, &factoryKey, factory)
}

func GetHandlerDescriptorFactory(
	ctx HandleContext,
) HandlerDescriptorFactory {
	switch f := ctx.GetValue(&factoryKey).(type) {
	case HandlerDescriptorFactory: return f
	default: return nil
	}
}