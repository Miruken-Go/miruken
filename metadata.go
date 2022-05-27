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
		key() any
		suppress() bool
		newHandlerDescriptor(
			policySpecs policySpecBuilder,
			observer    HandlerDescriptorObserver,
		) (*HandlerDescriptor, error)
	}

	// HandlerTypeSpec represents Handler type specifications.
	HandlerTypeSpec struct {
		typ reflect.Type
	}

	// HandlerFuncSpec represents Handler function specifications.
	HandlerFuncSpec struct {
		fun reflect.Value
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
	return fmt.Sprintf("HandlerTypeSpec %v", s.typ)
}

func (s HandlerTypeSpec) key() any {
	return s.typ
}

func (s HandlerTypeSpec) suppress() bool {
	return s.typ.Implements(_suppressDispatchType)
}

func (s HandlerTypeSpec) newHandlerDescriptor(
	policySpecs policySpecBuilder,
	observer    HandlerDescriptorObserver,
) (descriptor *HandlerDescriptor, invalid error) {
	typ        := s.typ
	bindings   := make(policyBindingsMap)
	descriptor  = &HandlerDescriptor{spec: s}

	var ctorSpec *policySpec
	var ctorPolicies []Policy
	var constructor *reflect.Method
	// Add constructor implicitly
	if ctor, ok := typ.MethodByName("Constructor"); ok {
		constructor = &ctor
		ctorType   := ctor.Type
		if ctorType.NumIn() > 1 {
			if spec, err := policySpecs.buildSpec(ctorType.In(1)); err == nil {
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
			if ctor, err := binder.NewConstructorBinding(typ, constructor, ctorSpec); err == nil {
				if observer != nil {
					observer.NotifyHandlerBinding(descriptor, ctor)
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
		if spec, err := policySpecs.buildSpec(methodType.In(1)); err == nil {
			if spec == nil { // not a handler method
				continue
			}
			for _, policy := range spec.policies {
				if binder, ok := policy.(MethodBinder); ok {
					if binding, errBind := binder.NewMethodBinding(method, spec); binding != nil {
						if observer != nil {
							observer.NotifyHandlerBinding(descriptor, binding)
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

func (s HandlerFuncSpec) Func() reflect.Value {
	return s.fun
}

func (s HandlerFuncSpec) String() string {
	return fmt.Sprintf("HandlerFuncSpec %v", s.fun)
}

func (s HandlerFuncSpec) key() any {
	return s.fun.Pointer()
}

func (s HandlerFuncSpec) suppress() bool {
	return false
}

func (s HandlerFuncSpec) newHandlerDescriptor(
	policySpecs policySpecBuilder,
	observer    HandlerDescriptorObserver,
) (descriptor *HandlerDescriptor, invalid error) {
	funType    := s.fun.Type()
	bindings   := make(policyBindingsMap)
	descriptor  = &HandlerDescriptor{spec: s}

	if funType.NumIn() < 1 {
		invalid = fmt.Errorf("missing callback spec in first argument")
	} else if spec, err := policySpecs.buildSpec(funType.In(0)); err == nil {
		if spec == nil {
			invalid = fmt.Errorf("first argument is not a callback spec")
		} else {
			for _, policy := range spec.policies {
				if binder, ok := policy.(FuncBinder); ok {
					if binding, errBind := binder.NewFuncBinding(s.fun, spec); binding != nil {
						if observer != nil {
							observer.NotifyHandlerBinding(descriptor, binding)
						}
						bindings.forPolicy(policy).insert(binding)
					} else if errBind != nil {
						invalid = multierror.Append(invalid, errBind)
					}
				} else {
					invalid = multierror.Append(invalid, fmt.Errorf(
						"policy %#v does not support function bindings", policy))
				}
			}
		}
	} else {
		invalid = multierror.Append(invalid, err)
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
	policy   Policy,
	handler  any,
	callback Callback,
	greedy   bool,
	composer Handler,
	guard    CallbackGuard,
) (result HandleResult) {
	if pb, found := d.bindings[policy]; found {
		key := callback.Key()
		return pb.reduce(key, func (
			binding Binding,
			result  HandleResult,
		) (HandleResult, bool) {
			if result.stop || (result.handled && !greedy) {
				return result, true
			}
			if matches, _ := policy.MatchesKey(binding.Key(), key, binding.Strict()); matches {
				if guard != nil {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve { return result, false }
				}
				if guard, ok := callback.(CallbackGuard); ok {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve { return result, false }
				}
				var filters []providedFilter
				if check, ok := callback.(interface{
					CanFilter() bool
				}); !ok || check.CanFilter() {
					var tp []FilterProvider
					if tf, ok := handler.(Filter); ok {
						tp = []FilterProvider{
							&FilterInstanceProvider{[]Filter{tf}, true},
						}
					}
					if providedFilters, err := orderedFilters(
						composer, binding, callback, binding.Filters(),
						d.Filters(), policy.Filters(), tp);
						providedFilters != nil && err == nil {
						filters = providedFilters
					} else {
						return result, false
					}
				}
				var out []any
				var err error
				context := HandleContext{
					handler,
					callback,
					binding,
					composer,
					greedy,
				}
				if len(filters) == 0 {
					out, err = binding.Invoke(context)
				} else {
					out, err = pipeline(context, filters, func(ctx HandleContext) ([]any, error) {
						return binding.Invoke(ctx)
					})
				}
				if err == nil {
					res, accepted := policy.AcceptResults(out)
					if res != nil {
						if accepted.handled {
							accepted = accepted.And(callback.ReceiveResult(res, binding.Strict(), composer))
						}
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

	// HandlerDescriptorObserver observes HandlerDescriptor Binding's.
	HandlerDescriptorObserver interface {
		NotifyHandlerBinding(
			descriptor *HandlerDescriptor,
			binding     Binding,
		)
	}
	HandlerDescriptorVisitorFunc func(*HandlerDescriptor, Binding)
)

func (f HandlerDescriptorVisitorFunc) NotifyHandlerBinding(
	descriptor *HandlerDescriptor,
	binding    Binding,
) {
	f(descriptor, binding)
}

// mutableDescriptorFactory creates HandlerDescriptor's on demand.
type mutableDescriptorFactory struct {
	sync.RWMutex
	policySpecBuilder
	descriptors map[any]*HandlerDescriptor
	observer    HandlerDescriptorObserver
}

func (f *mutableDescriptorFactory) MakeHandlerSpec(
	spec any,
) HandlerSpec {
	if IsNil(spec) {
		panic("spec cannot be nil")
	}

	var hs HandlerSpec

	switch h := spec.(type) {
	case HandlerSpec:
		hs = h
	case reflect.Type:
		hs = HandlerTypeSpec{h}
	default:
		typ := reflect.TypeOf(spec)
		if typ.Kind() == reflect.Func {
			hs = HandlerFuncSpec{reflect.ValueOf(spec)}
		} else {
			hs = HandlerTypeSpec{typ}
		}
	}

	if hs.suppress() {
		return nil
	}
	return hs
}

func (f *mutableDescriptorFactory) DescriptorOf(
	spec any,
) *HandlerDescriptor {
	handler := f.MakeHandlerSpec(spec)
	if handler == nil {
		return nil
	}
	f.RLock()
	defer f.RUnlock()
	return f.descriptors[handler.key()]
}

func (f *mutableDescriptorFactory) RegisterHandler(
	spec any,
) (*HandlerDescriptor, bool, error) {
	handler := f.MakeHandlerSpec(spec)
	if handler == nil {
		return nil, false, nil
	}

	f.Lock()
	defer f.Unlock()

	key := handler.key()
	if descriptor := f.descriptors[key]; descriptor != nil {
		return descriptor, false, nil
	}
	if descriptor, err := handler.newHandlerDescriptor(f.policySpecBuilder, f.observer); err == nil {
		f.descriptors[key] = descriptor
		return descriptor, true, nil
	} else {
		return nil, false, err
	}
}

// HandlerDescriptorFactoryBuilder build the HandlerDescriptorFactory.
type HandlerDescriptorFactoryBuilder struct {
	bindings []bindingBuilder
	observer HandlerDescriptorObserver
}

func (b *HandlerDescriptorFactoryBuilder) AddBindingBuilders(
	bindings ...bindingBuilder,
) *HandlerDescriptorFactoryBuilder {
	b.bindings = append(b.bindings, bindings...)
	return b
}

func (b *HandlerDescriptorFactoryBuilder) SetObserver(
	observer HandlerDescriptorObserver,
) *HandlerDescriptorFactoryBuilder {
	b.observer = observer
	return b
}

func (b *HandlerDescriptorFactoryBuilder) Build() HandlerDescriptorFactory {
	factory := &mutableDescriptorFactory{
		descriptors: make(map[any]*HandlerDescriptor),
		observer: b.observer,
	}
	bindings := make([]bindingBuilder, len(b.bindings)+4)
	bindings[0] = &factory.policySpecBuilder
	bindings[1] = bindingBuilderFunc(bindOptions)
	bindings[2] = bindingBuilderFunc(bindFilters)
	bindings[3] = bindingBuilderFunc(bindConstraints)
	for i, binding := range b.bindings {
		bindings[i+4] = binding
	}
	factory.policySpecBuilder.bindings = bindings
	return factory
}

type HandlerDescriptorFactoryOption interface {
	Configure(builder *HandlerDescriptorFactoryBuilder)
}
type ConfigureHandlerDescriptorFactory func(builder *HandlerDescriptorFactoryBuilder)

func (f ConfigureHandlerDescriptorFactory) Configure(
	builder *HandlerDescriptorFactoryBuilder,
) { f(builder) }

func NewMutableHandlerDescriptorFactory(
	opts ...HandlerDescriptorFactoryOption,
) HandlerDescriptorFactory {
	builder := &HandlerDescriptorFactoryBuilder{}
	for _, opt := range opts {
		opt.Configure(builder)
	}
	return builder.Build()
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
