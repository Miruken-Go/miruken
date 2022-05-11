package miruken

import (
	"container/list"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"sync"
)

type (
	// policyBindings maintains Binding's for a Policy.
	policyBindings struct {
		policy    Policy
		variant   list.List
		index     map[any]*list.Element
		invariant map[any][]Binding
	}

	// policyBindingsMap maps Policy's to policyBindings.
	policyBindingsMap map[Policy]*policyBindings
)

func (p *policyBindings) insert(binding Binding) {
	key := binding.Key()
	if variant, unknown := p.policy.IsVariantKey(key); variant {
		if unknown {
			p.variant.PushBack(binding)
			return
		}
		indexedElem := p.index[key]
		insert := indexedElem
		if insert == nil {
			insert = p.variant.Front()
		}
		for insert != nil && !p.policy.Less(binding, insert.Value.(Binding)) {
			insert = insert.Next()
		}
		var elem *list.Element
		if insert != nil {
			elem = p.variant.InsertBefore(binding, insert)
		} else {
			elem = p.variant.PushBack(binding)
		}
		if indexedElem == nil {
			p.index[key] = elem
		}
	} else {
		if p.invariant == nil {
			p.invariant = make(map[any][]Binding)
			p.invariant[key] = []Binding{binding}
		} else {
			bindings := append(p.invariant[key], binding)
			p.invariant[key] = bindings
		}
	}
}

func (p *policyBindings) reduce(
	key     any,
	reducer BindingReducer,
) (result HandleResult) {
	if reducer == nil {
		panic("reducer cannot be nil")
	}
	done := false
	result = NotHandled
	if variant, _ := p.policy.IsVariantKey(key); variant {
		elem := p.index[key]
		if elem == nil {
			elem = p.variant.Front()
		}
		for elem != nil {
			if result, done = reducer(elem.Value.(Binding), result); done {
				break
			}
			elem = elem.Next()
		}
	} else if p.invariant != nil {
		if bs := p.invariant[key]; bs != nil {
			for _, b := range bs {
				result, done = reducer(b, result)
				if done { break }
			}
		}
	}
	return result
}

func (p policyBindingsMap) forPolicy(policy Policy) *policyBindings {
	bindings, found := p[policy]
	if !found {
		bindings = &policyBindings{
			policy: policy,
			index:  make(map[any]*list.Element),
		}
		p[policy] = bindings
	}
	return bindings
}

type (
	// HandlerDescriptor describes the Binding's of a Handler.
	HandlerDescriptor struct {
		FilteredScope
		spec     HandlerSpec
		bindings policyBindingsMap
	}

	// HandlerSpec manages a category of HandlerDescriptor's.
	HandlerSpec interface {
		suppress() bool
		newHandlerDescriptor(
			policySpecs policySpecBuilder,
			visitor     HandlerDescriptorVisitor,
		) (*HandlerDescriptor, error)
	}

	// HandlerTypeSpec represents typed Handler specifications.
	HandlerTypeSpec struct {
		typ reflect.Type
	}

	// HandlerDescriptorError reports a failed HandlerDescriptor.
	HandlerDescriptorError struct {
		spec   HandlerSpec
		Reason error
	}
)

func (s HandlerTypeSpec) Type() reflect.Type {
	return s.typ
}

func (s HandlerTypeSpec) Name() string {
	if typ := s.typ; typ.Kind() == reflect.Ptr {
		return typ.Elem().Name()
	} else {
		return typ.Name()
	}
}

func (s HandlerTypeSpec) String() string {
	return fmt.Sprintf("%v", s.Type())
}

func (s HandlerTypeSpec) suppress() bool {
	return s.typ.Implements(_suppressDispatchType)
}

func (s HandlerTypeSpec) newHandlerDescriptor(
	policySpecs policySpecBuilder,
	visitor     HandlerDescriptorVisitor,
) (descriptor *HandlerDescriptor, invalid error) {
	typ        := s.typ
	descriptor  = &HandlerDescriptor{spec: s}
	bindings   := make(policyBindingsMap)
	var ctorSpec *policySpec
	var ctorPolicies []Policy
	var constructor *reflect.Method
	// Add constructor implicitly
	if ctor, ok := typ.MethodByName("Constructor"); ok {
		constructor = &ctor
		ctorType   := ctor.Type
		if ctorType.NumIn() > 1 {
			if spec, err := policySpecs.BuildSpec(ctorType.In(1)); err == nil {
				if spec != nil {
					ctorSpec     = spec
					ctorPolicies = spec.policies
				}
			} else {
				invalid = multierror.Append(invalid, err)
			}
		}
	}
	if _, noImplicit := typ.MethodByName("NoConstructor"); !noImplicit {
		addProvides := true
		for _, ctorPolicy := range ctorPolicies {
			if _, ok := ctorPolicy.(*providesPolicy); ok {
				addProvides = false
				break
			}
		}
		if addProvides {
			ctorPolicies = append(ctorPolicies, _providesPolicy)
		}
	} else if constructor != nil {
		invalid = multierror.Append(invalid, fmt.Errorf(
			"handler %v has both a Constructor and NoConstructor method", typ))
	}
	for _, ctorPolicy := range ctorPolicies {
		if binder, ok := ctorPolicy.(ConstructorBinder); ok {
			if ctor, err := binder.NewConstructorBinding(
				typ, constructor, ctorSpec); err == nil {
				if visitor != nil {
					visitor.VisitHandlerBinding(descriptor, ctor)
				}
				bindings.forPolicy(ctorPolicy).insert(ctor)
			} else {
				invalid = multierror.Append(invalid, err)
			}
		}
	}
	// Add callback builder explicitly
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name == "Constructor" || method.Name == "NoConstructor" {
			continue
		}
		methodType := method.Type
		if methodType.NumIn() < 2 {
			continue // must have a callback/spec
		}
		if spec, err := policySpecs.BuildSpec(methodType.In(1)); err == nil {
			if spec == nil { // not a handler ctor
				continue
			}
			for _, policy := range spec.policies {
				if binder, ok := policy.(MethodBinder); ok {
					if binding, errBind := binder.NewMethodBinding(method, spec); binding != nil {
						if visitor != nil {
							visitor.VisitHandlerBinding(descriptor, binding)
						}
						bindings.forPolicy(policy).insert(binding)
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
		return nil, &HandlerDescriptorError{s, invalid}
	}
	descriptor.bindings = bindings
	return descriptor, nil
}

func (e *HandlerDescriptorError) HandlerSpec() HandlerSpec {
	return e.spec
}

func (e *HandlerDescriptorError) Error() string {
	return fmt.Sprintf("invalid handler: %v reason: %v", e.spec, e.Reason)
}

func (e *HandlerDescriptorError) Unwrap() error { return e.Reason }

func (d *HandlerDescriptor) Dispatch(
	policy      Policy,
	handler     any,
	callback    any,
	rawCallback Callback,
	greedy      bool,
	composer    Handler,
) (result HandleResult) {
	if pb, found := d.bindings[policy]; found {
		key := rawCallback.Key()
		return pb.reduce(key, func (
			binding Binding,
			result  HandleResult,
		) (HandleResult, bool) {
			if result.stop || (result.handled && !greedy) {
				return result, true
			}
			if matches, _ := policy.MatchesKey(binding.Key(), key, binding.Strict()); matches {
				if guard, ok := rawCallback.(CallbackGuard); ok {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve { return result, false }
				}
				var filters []providedFilter
				if check, ok := rawCallback.(interface{
					CanFilter() bool
				}); !ok || check.CanFilter() {
					var tp []FilterProvider
					if tf, ok := handler.(Filter); ok {
						tp = []FilterProvider{
							&FilterInstanceProvider{[]Filter{tf}, true},
						}
					}
					if providedFilters, err := orderedFilters(
						composer, binding, rawCallback, binding.Filters(),
						d.Filters(), policy.Filters(), tp);
						providedFilters != nil && err == nil {
						filters = providedFilters
					} else {
						return result, false
					}
				}
				var out []any
				var err error
				context := HandleContext{callback, rawCallback, binding, composer}
				if len(filters) == 0 {
					out, err = binding.Invoke(context, handler)
				} else {
					out, err = pipeline(context, filters, func(ctx HandleContext) ([]any, error) {
						return binding.Invoke(ctx, handler)
					})
				}
				if err == nil {
					res, accepted := policy.AcceptResults(out)
					if res != nil {
						accepted = accepted.And(rawCallback.ReceiveResult(res, binding.Strict(), composer))
					}
					result = result.Or(accepted)
				} else {
					switch err.(type) {
					case RejectedError:
					case NotHandledError:
					case UnresolvedArgError:
						break
					default:
						result = result.WithError(err)
					}
				}
			}
			return result, result.stop || (result.handled && !greedy)
		})
	}
	return NotHandled
}

type (
	// HandlerDescriptorProvider returns Handler descriptors.
	HandlerDescriptorProvider interface {
		DescriptorOf(spec any) *HandlerDescriptor
	}

	// HandlerDescriptorFactory registers HandlerDescriptor's.
	HandlerDescriptorFactory interface {
		HandlerDescriptorProvider
		MakeHandlerSpec(spec any) HandlerSpec
		RegisterHandler(spec any) (*HandlerDescriptor, bool, error)
	}

	// HandlerDescriptorVisitor observes HandlerDescriptor Binding's.
	HandlerDescriptorVisitor interface {
		VisitHandlerBinding(
			descriptor *HandlerDescriptor,
			binding     Binding,
		)
	}
	HandlerDescriptorVisitorFunc func(*HandlerDescriptor, Binding)
)

func (f HandlerDescriptorVisitorFunc) VisitHandlerBinding(
	descriptor *HandlerDescriptor,
	binding    Binding,
) {
	f(descriptor, binding)
}

// mutableDescriptorFactory creates HandlerDescriptor's on demand.
type mutableDescriptorFactory struct {
	sync.RWMutex
	policySpecBuilder
	descriptors map[HandlerSpec]*HandlerDescriptor
	visitor     HandlerDescriptorVisitor
}

func (f *mutableDescriptorFactory) MakeHandlerSpec(
	spec any,
) HandlerSpec {
	if IsNil(spec) {
		panic("spec cannot be nil")
	}
	switch h := spec.(type) {
	case HandlerSpec:
		return h
	case reflect.Type:
		return HandlerTypeSpec{h}
	default:
		typ := reflect.TypeOf(spec)
		if typ.Kind() == reflect.Func {
			panic("handler func not supported yet")
		} else {
			return HandlerTypeSpec{typ}
		}
	}
}

func (f *mutableDescriptorFactory) DescriptorOf(
	spec any,
) *HandlerDescriptor {
	handler := f.MakeHandlerSpec(spec)
	f.RLock()
	defer f.RUnlock()
	return f.descriptors[handler]
}

func (f *mutableDescriptorFactory) RegisterHandler(
	spec any,
) (*HandlerDescriptor, bool, error) {
	handler := f.MakeHandlerSpec(spec)

	f.Lock()
	defer f.Unlock()

	if descriptor := f.descriptors[handler]; descriptor != nil {
		return descriptor, false, nil
	}
	if descriptor, err := handler.newHandlerDescriptor(f.policySpecBuilder, f.visitor); err == nil {
		f.descriptors[handler] = descriptor
		return descriptor, true, nil
	} else {
		return nil, false, err
	}
}

type MutableHandlerDescriptorFactoryOption interface {
	applyMutableFactoryOption(factory *mutableDescriptorFactory)
}

type mutableFactoryOptionFunc func(*mutableDescriptorFactory)

func (f mutableFactoryOptionFunc) applyMutableFactoryOption(
	factory *mutableDescriptorFactory,
) { f(factory) }

func NewMutableHandlerDescriptorFactory(
	opts ...MutableHandlerDescriptorFactoryOption,
) HandlerDescriptorFactory {
	factory := &mutableDescriptorFactory{
		descriptors: make(map[HandlerSpec]*HandlerDescriptor),
	}
	for _, opt := range opts {
		opt.applyMutableFactoryOption(factory)
	}
	return factory
}

func WithHandlerDescriptorVisitor(
	visitor HandlerDescriptorVisitor,
) MutableHandlerDescriptorFactoryOption {
	return mutableFactoryOptionFunc(func (factory *mutableDescriptorFactory) {
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
	callback any,
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

func (g *getHandlerDescriptorFactory) SuppressDispatch() {}

var _suppressDispatchType = TypeOf[suppressDispatch]()
