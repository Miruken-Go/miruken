package miruken

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// HandlerInfo describes the structure of a Handler.
	HandlerInfo struct {
		FilteredScope
		spec     HandlerSpec
		bindings policyInfoMap
		compound filterBindingGroup
	}

	// HandlerSpec is Factory for HandlerInfo and associated metadata.
	HandlerSpec interface {
		fmt.Stringer
		PkgPath() string
		key() any
		suppress() bool
		describe(
			builder bindingSpecFactory,
			observers []HandlerInfoObserver,
		) (*HandlerInfo, error)
	}

	// TypeSpec creates a HandlerInfo using all the exported
	// methods of reflect.Type instance.
	TypeSpec struct {
		typ reflect.Type
	}

	// FuncSpec creates a HandlerInfo from a single function.
	FuncSpec struct {
		fun reflect.Value
	}

	// HandlerInfoError reports a failed HandlerInfo.
	HandlerInfoError struct {
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
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

func (s TypeSpec) PkgPath() string {
	typ := s.typ
	if typ.Kind() == reflect.Ptr {
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

func (s TypeSpec) describe(
	factory   bindingSpecFactory,
	observers []HandlerInfoObserver,
) (info *HandlerInfo, invalid error) {
	typ := s.typ
	bindings := make(policyInfoMap)
	info = &HandlerInfo{spec: s}
	isFilter := typ.Implements(filterType)

	var ctorSpec     *bindingSpec
	var ctorPolicies []policyKey
	var ctor         *reflect.Method
	var inits        []*reflect.Method

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
			invalid = multierror.Append(invalid, err)
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
		invalid = multierror.Append(invalid, fmt.Errorf(
			"handler %v has both a Constructor and NoConstructor method", typ))
	}

	// Discover explicit callback handlers
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
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
						invalid = multierror.Append(invalid, err)
					} else if fb != nil {
						info.compound = append(info.compound, *fb)
					}
				}
				continue
			}
			for _, pk := range spec.policies {
				policy := pk.policy
				if binder, ok := policy.(MethodBinder); ok {
					if binding, err := binder.NewMethodBinding(&method, spec, pk.key); binding != nil {
						for _, observer := range observers {
							observer.BindingCreated(policy, info, binding)
						}
						bindings.forPolicy(policy).insert(policy, binding)
					} else if err != nil {
						invalid = multierror.Append(invalid, err)
					}
				}
			}
		} else {
			invalid = multierror.Append(invalid, err)
		}
	}

	// Build ctor bindings
	for _, ctorPk := range ctorPolicies {
		policy := ctorPk.policy
		if binder, ok := policy.(ConstructorBinder); ok {
			if ctor, err := binder.NewCtorBinding(typ, ctor, inits, ctorSpec, ctorPk.key); err == nil {
				for _, observer := range observers {
					observer.BindingCreated(policy, info, ctor)
				}
				bindings.forPolicy(policy).insert(policy, ctor)
			} else {
				invalid = multierror.Append(invalid, err)
			}
		}
	}

	if invalid != nil {
		return nil, &HandlerInfoError{s, invalid}
	}
	info.bindings = bindings
	return info, nil
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

func (s FuncSpec) describe(
	factory   bindingSpecFactory,
	observers []HandlerInfoObserver,
) (info *HandlerInfo, invalid error) {
	funType := s.fun.Type()
	bindings := make(policyInfoMap)
	info = &HandlerInfo{spec: s}

	if spec, err := factory.createSpec(funType, 1); err == nil {
		if spec == nil {
			invalid = fmt.Errorf("first argument is not a callback spec")
		} else {
			for _, pk := range spec.policies {
				policy := pk.policy
				if binder, ok := policy.(FuncBinder); ok {
					if binding, errBind := binder.NewFuncBinding(s.fun, spec, pk.key); binding != nil {
						for _, observer := range observers {
							observer.BindingCreated(policy, info, binding)
						}
						bindings.forPolicy(policy).insert(policy, binding)
					} else if errBind != nil {
						invalid = multierror.Append(invalid, errBind)
					}
				} else {
					invalid = multierror.Append(invalid, fmt.Errorf(
						"policy %T does not support function bindings", policy))
				}
			}
		}
	} else {
		invalid = multierror.Append(invalid, err)
	}
	if invalid != nil {
		return nil, &HandlerInfoError{s, invalid}
	}
	info.bindings = bindings
	return info, nil
}

// HandlerInfoError

func (e *HandlerInfoError) Error() string {
	return fmt.Sprintf("invalid handler: %v cause: %v", e.Spec, e.Cause)
}

func (e *HandlerInfoError) Unwrap() error {
	return e.Cause
}

// HandlerInfo

func (h *HandlerInfo) Spec() HandlerSpec {
	return h.spec
}

func (h *HandlerInfo) Dispatch(
	policy   Policy,
	handler  any,
	callback Callback,
	greedy   bool,
	composer Handler,
	guard    CallbackGuard,
) (result HandleResult) {
	if pb, found := h.bindings[policy]; found {
		key := callback.Key()
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
						composer, binding, callback, binding.Filters(),
						h.Filters(), policy.Filters(), tp); orderedFilters != nil && err == nil {
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
					Composer: composer,
					Greedy:   greedy,
				}
				if len(filters) == 0 {
					out, pout, err = binding.Invoke(ctx)
				} else {
					out, pout, err = pipeline(ctx, filters,
						func(ctx HandleContext) ([]any, *promise.Promise[[]any], error) {
							return binding.Invoke(ctx)
						})
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
	policy  Policy,
	ctx     *HandleContext,
	await   bool,
) (any, HandleResult) {
	res, accept, intents := policy.AcceptResults(results)
	if intents != nil && accept.Handled() && !accept.IsError() {
		pi, err := processIntents(intents, ctx, await)
		if err != nil {
			accept = accept.And(NotHandled).WithError(err)
		} else if pi != nil {
			return promise.Return(pi, res), accept
		}
	}
	return res, accept
}

func processIntents(
	intents []Intent,
	ctx     *HandleContext,
	await   bool,
) (*promise.Promise[struct{}], error) {
	var ps []*promise.Promise[any]
	for _, intent := range intents {
		if pi, err := intent.Apply(*ctx); err != nil {
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
	// HandlerInfoProvider returns HandlerInfo's.
	HandlerInfoProvider interface {
		Get(src any) *HandlerInfo
	}

	// HandlerInfoFactory registers HandlerInfo's.
	HandlerInfoFactory interface {
		HandlerInfoProvider
		Spec(src any) HandlerSpec
		Register(src any) (*HandlerInfo, bool, error)
	}

	// HandlerInfoObserver observes HandlerInfo creation.
	HandlerInfoObserver interface {
		BindingCreated(
			policy      Policy,
			handlerInfo *HandlerInfo,
			binding     Binding,
		)
		HandlerInfoCreated(handlerInfo *HandlerInfo)
	}
	HandlerInfoObserverFunc func(Policy, *HandlerInfo, Binding)
)

func (f HandlerInfoObserverFunc) BindingCreated(
	policy      Policy,
	handlerInfo *HandlerInfo,
	binding     Binding,
) {
	f(policy, handlerInfo, binding)
}

// mutableHandlerFactory creates HandlerInfo's on demand.
type mutableHandlerFactory struct {
	bindingSpecFactory
	handlers  map[any]*HandlerInfo
	observers []HandlerInfoObserver
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
) *HandlerInfo {
	spec := f.Spec(src)
	if spec == nil {
		return nil
	}
	return f.handlers[spec.key()]
}

func (f *mutableHandlerFactory) Register(
	src any,
) (*HandlerInfo, bool, error) {
	spec := f.Spec(src)
	if spec == nil {
		return nil, false, nil
	}
	key := spec.key()
	if info := f.handlers[key]; info != nil {
		return info, false, nil
	}
	if info, err := spec.describe(f.bindingSpecFactory, f.observers); err == nil {
		for _, observer := range f.observers {
			observer.HandlerInfoCreated(info)
		}
		f.handlers[key] = info
		return info, true, nil
	} else {
		return nil, false, err
	}
}

// HandlerInfoFactoryBuilder build the HandlerInfoFactory.
type HandlerInfoFactoryBuilder struct {
	parsers   []BindingParser
	observers []HandlerInfoObserver
}

func (b *HandlerInfoFactoryBuilder) Parsers(
	parsers ...BindingParser,
) *HandlerInfoFactoryBuilder {
	b.parsers = append(b.parsers, parsers...)
	return b
}

func (b *HandlerInfoFactoryBuilder) Observers(
	observers ...HandlerInfoObserver,
) *HandlerInfoFactoryBuilder {
	b.observers = append(b.observers, observers...)
	return b
}

func (b *HandlerInfoFactoryBuilder) Build() HandlerInfoFactory {
	factory := &mutableHandlerFactory{
		handlers:  make(map[any]*HandlerInfo),
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

// CurrentHandlerInfoFactory retrieves the current HandlerInfoFactory
// assigned to the Handler context.
func CurrentHandlerInfoFactory(
	handler Handler,
) HandlerInfoFactory {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	request := &CurrentHandlerInfoFactoryProvider{}
	handler.Handle(request, false, handler)
	return request.Factory
}

// CurrentHandlerInfoFactoryProvider Resolves the current HandlerInfoFactory
type CurrentHandlerInfoFactoryProvider struct {
	Factory HandlerInfoFactory
}

func (f *CurrentHandlerInfoFactoryProvider) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.callback
	}
	if get, ok := callback.(*CurrentHandlerInfoFactoryProvider); ok {
		get.Factory = f.Factory
		return Handled
	}
	return NotHandled
}

func (f *CurrentHandlerInfoFactoryProvider) SuppressDispatch() {}

func (f *CurrentHandlerInfoFactoryProvider) CabBatch() bool {
	return false
}

var (
	suppressDispatchType = internal.TypeOf[suppressDispatch]()
	initSpecType         = internal.TypeOf[Init]()
)
