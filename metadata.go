package miruken

import (
	"container/list"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"sync"
)

type (
	policyBindings struct {
		order OrderBinding
		typed *list.List
		index map[reflect.Type]*list.Element
		invar map[interface{}][]Binding
	}

	BindingReducer func(
		binding Binding,
		result  HandleResult,
	) (HandleResult, bool)
)

func (p *policyBindings) insert(binding Binding) {
	constraint := binding.Constraint()
	if typ, ok := constraint.(reflect.Type); ok {
		if typ == _interfaceType {
			p.typed.PushBack(binding)
			return
		}
		indexedElem := p.index[typ]
		insert := indexedElem
		if insert == nil {
			insert = p.typed.Front()
		}
		for insert != nil && !p.order.Less(binding, insert.Value.(Binding)) {
			insert = insert.Next()
		}
		var elem *list.Element
		if insert != nil {
			elem = p.typed.InsertBefore(binding, insert)
		} else {
			elem = p.typed.PushBack(binding)
		}
		if indexedElem == nil {
			p.index[typ] = elem
		}
	} else {
		if p.invar == nil {
			p.invar = make(map[interface{}][]Binding)
			p.invar[constraint] = []Binding{binding}
		} else {
			bindings := append(p.invar[constraint], binding)
			p.invar[constraint] = bindings
		}
	}
}

func (p *policyBindings) reduce(
	constraint interface{},
	reduce     BindingReducer,
) (result HandleResult) {
	done := false
	result = NotHandled
	if typ, ok := constraint.(reflect.Type); ok {
		elem := p.index[typ]
		if elem == nil {
			elem = p.typed.Front()
		}
		for !done && elem != nil {
			result, done = reduce(elem.Value.(Binding), result)
			elem = elem.Next()
		}
	} else if p.invar != nil {
		if bs := p.invar[constraint]; bs != nil {
			for _, b := range bs {
				result, done = reduce(b, result)
				if done { break }
			}
		}
	}
	return result
}

func newPolicyBindings(order OrderBinding) *policyBindings {
	if order == nil {
		panic("order cannot be nil")
	}
	return &policyBindings{
		order,
		list.New(),
		make(map[reflect.Type]*list.Element),
		nil,
	}
}

// policyBindingsMap represents a mapping from Policy to Binding's.
type policyBindingsMap map[Policy]*policyBindings

func (p policyBindingsMap) getBindings(policy Policy) *policyBindings {
	policyBindings, found := p[policy]
	if !found {
		policyBindings = newPolicyBindings(policy)
		p[policy] = policyBindings
	}
	return policyBindings
}

// HandlerDescriptor maintains a list of Binding's for a Handler type.
type HandlerDescriptor struct {
	handlerType reflect.Type
	bindings    policyBindingsMap
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
	if pb, found := d.bindings[policy]; found {
		if constraint == nil {
			switch typ := callback.(type) {
			case reflect.Type: constraint = typ
			default: constraint = reflect.TypeOf(callback)
			}
		}
		return pb.reduce(constraint, func (
			binding Binding,
			result  HandleResult,
		) (HandleResult, bool) {
			if result.stop || (result.handled && !greedy) {
				return result, true
			}
			if binding.Matches(constraint, policy.Variance()) {
				if guard, ok := rawCallback.(CallbackGuard); ok {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve { return result, false }
				}
				if out, err := binding.Invoke(
						handler, callback, rawCallback, composer, results); err == nil {
					res, accepted := policy.AcceptResults(out)
					if accepted.IsHandled() && results != nil &&
						results.ReceiveResult(res, binding.Strict(), greedy, composer) {
						accepted = accepted.Or(Handled)
					}
					result = result.Or(accepted)
				}
			}
			return result, false
		})
	}
	return NotHandled
}

// HandlerDescriptorError reports a failed descriptor.
type HandlerDescriptorError struct {
	HandlerType reflect.Type
	Reason      error
}

func (e *HandlerDescriptorError) Error() string {
	return fmt.Sprintf("invalid handler: %v reason: %v", e.HandlerType, e.Reason)
}

func (e *HandlerDescriptorError) Unwrap() error { return e.Reason }

// HandlerDescriptorProvider return descriptors for the handler type.
type HandlerDescriptorProvider interface {
	GetHandlerDescriptor(
		handler reflect.Type,
	) (*HandlerDescriptor, error)
}

// HandlerDescriptorFactory adds registration capability to the HandlerDescriptorProvider.
type HandlerDescriptorFactory interface {
	HandlerDescriptorProvider

	RegisterHandlerType(
		handlerType reflect.Type,
	) (*HandlerDescriptor, bool, error)
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

// mutableFactory create HandlerDescriptor's on demand.
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
) (*HandlerDescriptor, bool, error) {
	if err := validHandlerType(handlerType); err != nil {
		return nil, false, err
	}

	f.Lock()
	defer f.Unlock()

	if descriptor := f.descriptors[handlerType]; descriptor != nil {
		return descriptor, false, nil
	}
	if descriptor, err := f.newHandlerDescriptor(handlerType); err == nil {
		f.descriptors[handlerType] = descriptor
		return descriptor, true, nil
	} else {
		return nil, false, err
	}
}

func (f *mutableFactory) newHandlerDescriptor(
	handlerType reflect.Type,
) (descriptor *HandlerDescriptor, invalid error) {
	descriptor = &HandlerDescriptor{
		handlerType: handlerType,
	}
	bindings := make(policyBindingsMap)
	// Add constructors implicitly
	if handlerType.Kind() == reflect.Ptr {
		provides := ProvidesPolicy()
		policies := []Policy{ provides }
		var initMethod *reflect.Method
		var initSpec *policySpec
		if method, ok := handlerType.MethodByName("Initialize"); ok {
			initMethod = &method
			initMethodType := initMethod.Type
			if initMethodType.NumIn() > 1 {
				if spec, err := buildPolicySpec(initMethodType.In(1)); err == nil {
					if spec != nil {
						initSpec = spec
						for _, policy := range spec.policies {
							if policy != provides {
								policies = append(policies, policy)
							}
						}
					}
				} else {
					invalid = multierror.Append(invalid, err)
				}
			}
		}
		for _, policy := range policies {
			if binder, ok := provides.(constructorBinder); ok {
				if ctor, err := binder.newConstructorBinding(
					handlerType, initMethod, initSpec); err == nil {
					if f.visitor != nil {
						f.visitor.VisitHandlerBinding(descriptor, ctor)
					}
					bindings.getBindings(policy).insert(ctor)
				} else {
					invalid = multierror.Append(invalid, err)
				}
			}
		}
	}
	// Add callback types explicitly
	for i := 0; i < handlerType.NumMethod(); i++ {
		method     := handlerType.Method(i)
		if method.Name == "Initialize" {
			continue
		}
		methodType := method.Type
		if methodType.NumIn() < 2 {
			continue // must have a policy/spec
		}
		if spec, err := buildPolicySpec(methodType.In(1)); err == nil {
			if spec == nil { // not a handler method
				continue
			}
			for _, policy := range spec.policies {
				if binder, ok := policy.(methodBinder); ok {
					if binding, errBind := binder.newMethodBinding(method, spec); binding != nil {
						if f.visitor != nil {
							f.visitor.VisitHandlerBinding(descriptor, binding)
						}
						bindings.getBindings(policy).insert(binding)
					} else if errBind != nil {
						invalid = multierror.Append(invalid, errBind)
					}
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

func GetHandlerDescriptorFactory(
	handler Handler,
) HandlerDescriptorFactory {
	if handler == nil {
		panic("handler cannot be nil")
	}
	get := &getHandlerDescriptorFactory{}
	handler.Handle(get, false, handler)
	return get.factory
}

// getHandlerDescriptorFactory resolves the current HandlerDescriptorFactory
type getHandlerDescriptorFactory struct {
	factory HandlerDescriptorFactory
}

func (g *getHandlerDescriptorFactory) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.callback
	}
	if getFactory, ok := callback.(*getHandlerDescriptorFactory); ok {
		getFactory.factory = g.factory
		return Handled
	}
	return NotHandled
}

func (g *getHandlerDescriptorFactory) suppressDispatch() {}
