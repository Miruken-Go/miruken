package miruken

import (
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strings"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// HandlerRuntime provides Handler runtime support.
	HandlerRuntime struct {
		FilterScope
		spec     HandlerSpec
		filters  FilterScope
		bindings policyBindingMap
		compound filterBindingGroup
	}

	runtimeObserverMap map[runtimeObserverType][]HandlerRuntimeObserver

	// HandlerSpec is factory for HandlerRuntime and metadata.
	HandlerSpec interface {
		fmt.Stringer
		PkgPath() string
		key() any
		suppress() bool
		newRuntime(
			builder bindingSpecFactory,
			observers runtimeObserverMap,
		) (*HandlerRuntime, error)
	}

	// TypeSpec creates a HandlerRuntime using all the exported
	// methods of reflect.Type instance.
	TypeSpec struct {
		typ reflect.Type
	}

	// FuncSpec creates a HandlerRuntime from a single function.
	FuncSpec struct {
		fun reflect.Value
	}

	// HandlerRuntimeError reports a failed HandlerRuntime.
	HandlerRuntimeError struct {
		Spec  HandlerSpec
		Cause error
	}
)

// TypeSpec

func (s TypeSpec) Type() reflect.Type {
	return s.typ
}

func (s TypeSpec) Name() string {
	typ := s.typ
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.Name()
}

func (s TypeSpec) PkgPath() string {
	typ := s.typ
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.PkgPath()
}

func (s TypeSpec) String() string {
	return fmt.Sprintf("%v in %v package", s.typ, s.PkgPath())
}

func (s TypeSpec) key() any {
	return s.typ
}

func (s TypeSpec) suppress() bool {
	return s.typ.Implements(suppressDispatchType)
}

func (s TypeSpec) newRuntime(
	factory bindingSpecFactory,
	observers runtimeObserverMap,
) (runtime *HandlerRuntime, invalid error) {
	typ := s.typ
	bindings := make(policyBindingMap)
	runtime = &HandlerRuntime{spec: s}
	isFilter := typ.Implements(filterType)

	observers.notify(runtimeCreatedObserver, runtime, nil, nil)

	var ctorSpec *bindingSpec
	var ctorPolicies []policyKey
	var ctor *reflect.Method
	var inits []*reflect.Method

	// Add ctor implicitly
	if ctorMethod, ok := typ.MethodByName("Constructor"); ok {
		ctor = &ctorMethod
		ctorType := ctor.Type
		if spec, err := factory.createSpec(ctorType, 2); err == nil {
			if spec != nil {
				ctorSpec = spec
				ctorPolicies = spec.policies
			}
		} else {
			invalid = errors.Join(invalid, err)
		}
	}

	// Check for ctor suppression
	if _, noImplicit := typ.MethodByName("NoConstructor"); !noImplicit {
		addProvides := true
		for _, ctorPk := range ctorPolicies {
			if _, ok := ctorPk.policy.(*providesPolicy); ok {
				addProvides = false
				break
			}
		}
		if addProvides {
			ctorPolicies = append(ctorPolicies, policyKey{policy: providesPolicyIns})
		}
	} else if ctor != nil {
		invalid = errors.Join(invalid, fmt.Errorf(
			"handler %v has both a Constructor and NoConstructor method", typ))
	}

	// Discover method callback handlers
	for method := range typ.Methods() {
		if method.Name == "Constructor" || method.Name == "NoConstructor" {
			continue
		}
		methodType := method.Type
		if spec, err := factory.createSpec(methodType, 2); err == nil {
			if spec == nil { // not a handler method
				if strings.HasPrefix(method.Name, "Init") ||
					methodType.NumIn() >= 2 && methodType.In(1) == initSpecType {
					inits = append(inits, &method)
				} else if !isFilter {
					if fb, err := parseFilterMethod(&method); err != nil {
						invalid = errors.Join(invalid, err)
					} else if fb != nil {
						runtime.compound = append(runtime.compound, *fb)
					}
				}
				continue
			}
			for _, pk := range spec.policies {
				policy := pk.policy
				if binder, ok := policy.(MethodBinder); ok {
					if binding, err := binder.NewMethodBinding(&method, spec, pk.key); binding != nil {
						bindings.insert(policy, binding)
						observers.notify(runtimeBindingObserver, runtime, binding, policy)
					} else if err != nil {
						invalid = errors.Join(invalid, err)
					}
				}
			}
		} else {
			invalid = errors.Join(invalid, err)
		}
	}

	// Build ctor bindings
	// Must come after methods to collect Initializers.
	for _, ctorPk := range ctorPolicies {
		policy := ctorPk.policy
		if binder, ok := policy.(ConstructorBinder); ok {
			if ctor, err := binder.NewCtorBinding(typ, ctor, inits, ctorSpec, ctorPk.key); err == nil {
				bindings.insert(policy, ctor)
				observers.notify(runtimeBindingObserver, runtime, ctor, policy)
			} else {
				invalid = errors.Join(invalid, err)
			}
		}
	}

	if invalid != nil {
		return nil, &HandlerRuntimeError{s, invalid}
	}
	runtime.bindings = bindings
	return runtime, nil
}

// FuncSpec

func (s FuncSpec) Func() reflect.Value {
	return s.fun
}

func (s FuncSpec) String() string {
	return fmt.Sprintf("FuncSpec(%v)", s.fun)
}

func (s FuncSpec) PkgPath() string {
	return ""
}

func (s FuncSpec) key() any {
	return s.fun.Pointer()
}

func (s FuncSpec) suppress() bool {
	return false
}

func (s FuncSpec) newRuntime(
	factory bindingSpecFactory,
	observers runtimeObserverMap,
) (runtime *HandlerRuntime, invalid error) {
	funType := s.fun.Type()
	bindings := make(policyBindingMap)
	runtime = &HandlerRuntime{spec: s}

	observers.notify(runtimeCreatedObserver, runtime, nil, nil)

	if spec, err := factory.createSpec(funType, 1); err == nil {
		if spec == nil {
			invalid = fmt.Errorf("first argument is not a callback spec")
		} else {
			for _, pk := range spec.policies {
				policy := pk.policy
				if binder, ok := policy.(FuncBinder); ok {
					if binding, errBind := binder.NewFuncBinding(s.fun, spec, pk.key); binding != nil {
						bindings.insert(policy, binding)
						observers.notify(runtimeBindingObserver, runtime, binding, policy)
					} else if errBind != nil {
						invalid = errors.Join(invalid, errBind)
					}
				} else {
					invalid = errors.Join(invalid, fmt.Errorf(
						"policy %T does not support function bindings", policy))
				}
			}
		}
	} else {
		invalid = errors.Join(invalid, err)
	}
	if invalid != nil {
		return nil, &HandlerRuntimeError{s, invalid}
	}
	runtime.bindings = bindings
	return runtime, nil
}

// HandlerRuntimeError

func (e *HandlerRuntimeError) Error() string {
	return fmt.Sprintf("invalid handler: %v cause: %v", e.Spec, e.Cause)
}

func (e *HandlerRuntimeError) Unwrap() error {
	return e.Cause
}

// HandlerRuntime

func (h *HandlerRuntime) Spec() HandlerSpec {
	return h.spec
}

func (h *HandlerRuntime) Bindings() iter.Seq2[Policy, iter.Seq[Binding]] {
	return h.bindings.all()
}

func (h *HandlerRuntime) BindingsFor(policy Policy) iter.Seq[Binding] {
	return h.bindings.policy(policy)
}

func (h *HandlerRuntime) Dispatch(
	policy Policy,
	handler any,
	callback Callback,
	greedy bool,
	composer Handler,
	guard CallbackGuard,
) (result HandleResult) {
	if pb, found := h.bindings[policy]; found {
		key := callback.Key()
		var filterOpts FilterOptions
		var filterOptsResolved bool
		return pb.reduce(key, policy, func(
			binding Binding,
			result HandleResult,
		) (HandleResult, bool) {
			if result.stop || (result.handled && !greedy) {
				return result, true
			}
			if matches, _ := policy.MatchesKey(binding.Key(), key, false); matches {
				if guard != nil {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve {
						return result, false
					}
				}
				if guard, ok := callback.(CallbackGuard); ok {
					reset, approve := guard.CanDispatch(handler, binding)
					defer func() {
						if reset != nil {
							reset()
						}
					}()
					if !approve {
						return result, false
					}
				}
				var filters []providedFilter
				if check, ok := callback.(interface {
					CanFilter() bool
				}); !ok || check.CanFilter() {
					if !filterOptsResolved {
						filterOpts, _ = GetOptions[FilterOptions](composer)
						filterOptsResolved = true
					}
					var tp []FilterProvider
					// Only apply compound filters for handles policy
					if policy == handlesPolicyIns {
						if comp := h.compound; comp != nil {
							tp = []FilterProvider{
								&FilterInstanceProvider{[]Filter{
									compoundHandler{handler, comp},
								}, true},
							}
						} else if tf, ok := handler.(Filter); ok {
							tp = []FilterProvider{
								&FilterInstanceProvider{[]Filter{tf}, true},
							}
						}
					}
					if orderedFilters, err := orderFilters(
						filterOpts, binding, callback, composer,
						binding.Filters(), h.filters.Filters(), policy.Filters(), tp); orderedFilters != nil && err == nil {
						filters = orderedFilters
					} else {
						return result, false
					}
				}
				var out []any
				var pout *promise.Promise[[]any]
				var err error
				ctx := HandleContext{
					Handler:  handler,
					Callback: callback,
					Binding:  binding,
					Runtime:  h,
					Composer: composer,
					Greedy:   greedy,
				}
				if len(filters) == 0 {
					out, pout, err = binding.Invoke(ctx)
				} else {
					out, pout, err = pipelineInvoke(ctx, filters, binding)
				}
				if err == nil {
					if pout != nil {
						out = []any{promise.Then(pout, func(oo []any) any {
							res, accept := applyResults(oo, policy, &ctx, true)
							if accept.IsError() {
								panic(accept.Error())
							} else if !accept.Handled() {
								panic(&NotHandledError{callback})
							}
							return res
						})}
					}
					res, accept := applyResults(out, policy, &ctx, false)
					if !internal.IsNil(res) {
						if accept.handled {
							strict := policy.Strict() || binding.Strict()
							accept = accept.And(callback.ReceiveResult(res, strict, composer))
						}
					}
					result = result.Or(accept)
				} else {
					var rejectedError *RejectedError
					var notHandledError *NotHandledError
					var unresolvedArgError *UnresolvedArgError
					switch {
					case errors.As(err, &rejectedError):
					case errors.As(err, &notHandledError):
					case errors.As(err, &unresolvedArgError):
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

func applyResults(
	results []any,
	policy Policy,
	ctx *HandleContext,
	await bool,
) (any, HandleResult) {
	res, accept, effects, cascade := policy.AcceptResults(results)
	if len(cascade) > 0 {
		effects = append(effects, Cascade(cascade...))
	}
	if effects != nil && accept.Handled() && !accept.IsError() {
		pi, err := processEffects(effects, ctx, await)
		if err != nil {
			accept = accept.And(NotHandled).WithError(err)
		} else if pi != nil {
			return promise.Return(pi, res), accept
		}
	}
	return res, accept
}

func processEffects(
	effects []Effect,
	ctx *HandleContext,
	await bool,
) (*promise.Promise[struct{}], error) {
	var ps []*promise.Promise[any]
	for _, effect := range effects {
		if pi, err := effect.Apply(*ctx); err != nil {
			return nil, err
		} else if pi != nil {
			ps = append(ps, pi.Then(func(data any) any { return data }))
		}
	}
	switch len(ps) {
	case 0:
		return nil, nil
	case 1:
		x := ps[0]
		if await {
			_, err := x.Await()
			return nil, err
		}
		return promise.Erase(x), nil
	default:
		x := promise.All(nil, ps...)
		if await {
			_, err := x.Await()
			return nil, err
		}
		return promise.Erase(x), nil
	}
}

type (
	// HandlerRuntimeProvider returns a HandlerRuntime.
	HandlerRuntimeProvider interface {
		Get(src any) *HandlerRuntime
	}

	// HandlerRuntimeFactory registers a HandlerRuntime.
	HandlerRuntimeFactory interface {
		HandlerRuntimeProvider
		Spec(src any) HandlerSpec
		Register(src any) (*HandlerRuntime, bool, error)
	}

	// HandlerRuntimeObserver is a generic HandlerRuntime observer.
	HandlerRuntimeObserver = any

	// HandlerRuntimeCreatedObserver observes HandlerRuntime creation.
	// This observer is called after the HandlerRuntime has been created,
	// but not fully initialized with bindings.
	HandlerRuntimeCreatedObserver interface {
		HandlerRuntimeCreated(*HandlerRuntime)
	}
	HandlerRuntimeCreatedObserverFunc func(*HandlerRuntime)

	// HandlerRuntimeBindingObserver observes HandlerRuntime Binding creation.
	// This observer is called after each HandlerRuntime Binding has been added.
	HandlerRuntimeBindingObserver interface {
		HandlerRuntimeBinding(*HandlerRuntime, Binding, Policy)
	}
	HandlerRuntimeBindingObserverFunc func(*HandlerRuntime, Binding, Policy)

	// HandlerRuntimeRegisteredObserver observes HandlerRuntime registration.
	// This observer is called after the HandlerRuntime has been registered
	// and fully initialized with all bindings.
	HandlerRuntimeRegisteredObserver interface {
		HandlerRuntimeRegistered(*HandlerRuntime)
	}
	HandlerRuntimeRegisteredObserverFunc func(*HandlerRuntime)

	// mutableHandlerFactory creates HandlerRuntime on demand.
	mutableHandlerFactory struct {
		bindingSpecFactory
		handlers  map[any]*HandlerRuntime
		observers runtimeObserverMap
	}

	runtimeObserverType uint8
)

const (
	runtimeCreatedObserver = runtimeObserverType(1 << iota)
	runtimeBindingObserver
	runtimeRegisteredObserver
)

func (o runtimeObserverMap) register(
	observers ...HandlerRuntimeObserver,
) {
	for _, observer := range observers {
		if _, ok := observer.(HandlerRuntimeCreatedObserver); ok {
			o[runtimeCreatedObserver] = append(o[runtimeCreatedObserver], observer)
		}
		if _, ok := observer.(HandlerRuntimeBindingObserver); ok {
			o[runtimeBindingObserver] = append(o[runtimeBindingObserver], observer)
		}
		if _, ok := observer.(HandlerRuntimeRegisteredObserver); ok {
			o[runtimeRegisteredObserver] = append(o[runtimeRegisteredObserver], observer)
		}
	}
}

func (o runtimeObserverMap) notify(
	observerType runtimeObserverType,
	runtime *HandlerRuntime,
	binding Binding,
	policy Policy,
) {
	if o == nil {
		return
	}
	for _, observer := range o[observerType] {
		switch observerType {
		case runtimeCreatedObserver:
			observer.(HandlerRuntimeCreatedObserver).HandlerRuntimeCreated(runtime)
		case runtimeBindingObserver:
			observer.(HandlerRuntimeBindingObserver).HandlerRuntimeBinding(runtime, binding, policy)
		case runtimeRegisteredObserver:
			observer.(HandlerRuntimeRegisteredObserver).HandlerRuntimeRegistered(runtime)
		}
	}
}

func (f *mutableHandlerFactory) Spec(
	src any,
) HandlerSpec {
	if internal.IsNil(src) {
		panic("src cannot be nil")
	}
	var hs HandlerSpec
	switch h := src.(type) {
	case HandlerSpec:
		hs = h
	case reflect.Type:
		hs = TypeSpec{h}
	default:
		typ := reflect.TypeOf(src)
		if typ.Kind() == reflect.Func {
			hs = FuncSpec{reflect.ValueOf(src)}
		} else {
			hs = TypeSpec{typ}
		}
	}
	if hs.suppress() {
		return nil
	}
	return hs
}

func (f *mutableHandlerFactory) Get(
	src any,
) *HandlerRuntime {
	spec := f.Spec(src)
	if spec == nil {
		return nil
	}
	return f.handlers[spec.key()]
}

func (f *mutableHandlerFactory) Register(
	src any,
) (*HandlerRuntime, bool, error) {
	spec := f.Spec(src)
	if spec == nil {
		return nil, false, nil
	}
	key := spec.key()
	if runtime := f.handlers[key]; runtime != nil {
		return runtime, false, nil
	}
	if runtime, err := spec.newRuntime(f.bindingSpecFactory, f.observers); err == nil {
		f.handlers[key] = runtime
		f.observers.notify(runtimeRegisteredObserver, runtime, nil, nil)
		return runtime, true, nil
	} else {
		return nil, false, err
	}
}

func (f HandlerRuntimeCreatedObserverFunc) HandlerRuntimeCreated(
	runtime *HandlerRuntime,
) {
	f(runtime)
}

func (f HandlerRuntimeBindingObserverFunc) HandlerRuntimeBinding(
	runtime *HandlerRuntime,
	binding Binding,
	policy Policy,
) {
	f(runtime, binding, policy)
}

func (f HandlerRuntimeRegisteredObserverFunc) HandlerRuntimeRegistered(
	runtime *HandlerRuntime,
) {
	f(runtime)
}

// HandlerRuntimeFactoryBuilder builds the HandlerRuntimeFactory.
type HandlerRuntimeFactoryBuilder struct {
	parsers   []BindingParser
	observers runtimeObserverMap
}

func (b *HandlerRuntimeFactoryBuilder) Parsers(
	parsers ...BindingParser,
) *HandlerRuntimeFactoryBuilder {
	b.parsers = append(b.parsers, parsers...)
	return b
}

func (b *HandlerRuntimeFactoryBuilder) Observers(
	observers ...HandlerRuntimeObserver,
) *HandlerRuntimeFactoryBuilder {
	if len(observers) == 0 {
		return b
	}
	b.observers = make(runtimeObserverMap)
	b.observers.register(observers...)
	return b
}

func (b *HandlerRuntimeFactoryBuilder) Build() HandlerRuntimeFactory {
	factory := &mutableHandlerFactory{
		handlers:  make(map[any]*HandlerRuntime),
		observers: b.observers,
	}
	parsers := make([]BindingParser, len(b.parsers)+4)
	parsers[0] = &factory.bindingSpecFactory
	parsers[1] = BindingParserFunc(parseOptions)
	parsers[2] = BindingParserFunc(parseFilters)
	parsers[3] = BindingParserFunc(parseConstraints)
	for i, binding := range b.parsers {
		parsers[i+4] = binding
	}
	factory.bindingSpecFactory.parsers = parsers
	return factory
}

// CurrentHandlerRuntimeFactory retrieves the current
// HandlerRuntimeFactory assigned to the Handler context.
func CurrentHandlerRuntimeFactory(
	handler Handler,
) HandlerRuntimeFactory {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	request := &CurrentHandlerRuntimeFactoryProvider{}
	handler.Handle(request, false, handler)
	return request.Factory
}

// CurrentHandlerRuntimeFactoryProvider Resolves the current HandlerRuntimeFactory.
type CurrentHandlerRuntimeFactoryProvider struct {
	Factory HandlerRuntimeFactory
}

func (f *CurrentHandlerRuntimeFactoryProvider) Handle(
	callback any,
	greedy bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.callback
	}
	if get, ok := callback.(*CurrentHandlerRuntimeFactoryProvider); ok {
		get.Factory = f.Factory
		return Handled
	}
	return NotHandled
}

func (f *CurrentHandlerRuntimeFactoryProvider) SuppressDispatch() {}

func (f *CurrentHandlerRuntimeFactoryProvider) CabBatch() bool {
	return false
}

var (
	suppressDispatchType = reflect.TypeFor[suppressDispatch]()
	initSpecType         = reflect.TypeFor[Init]()
)
