package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// HandlerInfo describes the structure of a Handler.
	HandlerInfo struct {
		FilteredScope
		spec     HandlerSpec
		bindings policyInfoMap
	}

	// HandlerSpec is factory for HandlerInfo and associated metadata.
	HandlerSpec interface {
		fmt.Stringer
		PkgPath() string
		key() any
		suppress() bool
		describe(
			builder   bindingSpecFactory,
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
	typ      := s.typ
	bindings := make(policyInfoMap)
	info      = &HandlerInfo{spec: s}

	var ctorSpec *bindingSpec
	var ctorPolicies []policyKey
	var constructor *reflect.Method
	// Add constructor implicitly
	if ctor, ok := typ.MethodByName("Constructor"); ok {
		constructor = &ctor
		ctorType := ctor.Type
		if spec, err := factory.createSpec(ctorType, 2); err == nil {
			if spec != nil {
				ctorSpec     = spec
				ctorPolicies = spec.policies
			}
		} else {
			invalid = multierror.Append(invalid, err)
		}
	}
	if _, noImplicit := typ.MethodByName("NoConstructor"); !noImplicit {
		addProvides := true
		for _, ctorPk := range ctorPolicies {
			if _, ok := ctorPk.policy.(*providesPolicy); ok {
				addProvides = false
				break
			}
		}
		if addProvides {
			ctorPolicies = append(ctorPolicies, policyKey{policy: providesPolicyInstance})
		}
	} else if constructor != nil {
		invalid = multierror.Append(invalid, fmt.Errorf(
			"handler %v has both a Constructor and NoConstructor method", typ))
	}
	for _, ctorPk := range ctorPolicies {
		policy := ctorPk.policy
		if binder, ok := policy.(ConstructorBinder); ok {
			if ctor, err := binder.NewConstructorBinding(typ, constructor, ctorSpec, ctorPk.key); err == nil {
				for _, observer := range observers {
					observer.BindingCreated(policy, info, ctor)
				}
				bindings.forPolicy(policy).insert(policy, ctor)
			} else {
				invalid = multierror.Append(invalid, err)
			}
		}
	}
	// Add callback factory explicitly
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name == "Constructor" || method.Name == "NoConstructor" {
			continue
		}
		methodType := method.Type
		if spec, err := factory.createSpec(methodType, 2); err == nil {
			if spec == nil { // not a handler method
				continue
			}
			for _, pk := range spec.policies {
				policy := pk.policy
				if binder, ok := policy.(MethodBinder); ok {
					if binding, errBind := binder.NewMethodBinding(method, spec, pk.key); binding != nil {
						for _, observer := range observers {
							observer.BindingCreated(policy, info, binding)
						}
						bindings.forPolicy(policy).insert(policy, binding)
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
	funType    := s.fun.Type()
	bindings   := make(policyInfoMap)
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

func (d *HandlerInfo) Spec() HandlerSpec {
	return d.spec
}

func (d *HandlerInfo) Dispatch(
	policy   Policy,
	handler  any,
	callback Callback,
	greedy   bool,
	composer Handler,
	guard    CallbackGuard,
) (result HandleResult) {
	if pb, found := d.bindings[policy]; found {
		key := callback.Key()
		return pb.reduce(key, policy, func (
			binding Binding,
			result  HandleResult,
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
				var out  []any
				var pout *promise.Promise[[]any]
				var err  error
				ctx := HandleContext{
					Handler:  handler,
					Callback: callback,
					Binding:  binding,
					Composer: composer,
					Greedy:   greedy,
				}
				if len(filters) == 0 {
					out, pout, err = applySideEffects(binding, &ctx)
				} else {
					out, pout, err = pipeline(ctx, filters,
						func(ctx HandleContext) ([]any, *promise.Promise[[]any], error) {
							return applySideEffects(binding, &ctx)
					})
				}
				if err == nil {
					if pout != nil {
						out = []any{promise.Then(pout, func(oo []any) any {
							res, _ := policy.AcceptResults(oo)
							return res
						})}
					}
					res, accept := policy.AcceptResults(out)
					if res != nil {
						if accept.handled {
							strict := policy.Strict() || binding.Strict()
							accept = accept.And(callback.ReceiveResult(res, strict, composer))
						}
					}
					result = result.Or(accept)
				} else {
					switch err.(type) {
					case *RejectedError:
					case *NotHandledError:
					case *UnresolvedArgError:
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


func applySideEffects(
    binding Binding,
	ctx     *HandleContext,
) (out []any, pout *promise.Promise[[]any], err error) {
	out, pout, err = binding.Invoke(*ctx)
	if err != nil {
		return
	} else if pout != nil {
		pout = promise.Then(pout, func(oo []any) []any {
			oo, _, err = processSideEffects(oo, ctx, true)
			if err != nil {
				panic(err)
			}
			return oo
		})
	} else if len(out) > 0 {
		out, pout, err = processSideEffects(out, ctx, false)
	}
	return
}

func processSideEffects(
	out   []any,
	ctx   *HandleContext,
	await bool,
) ([]any, *promise.Promise[[]any], error) {
	temp := out[:0]
	var ps []*promise.Promise[any]
	for _, o := range out {
		if se, ok := o.(SideEffect); ok {
			if p, err := se.Apply(se, *ctx); err != nil {
				return nil, nil, err
			} else if p != nil {
				ps = append(ps, p.Then(func(data any) any { return data }))
			}
		} else {
			temp = append(temp, o)
		}
		out = temp
	}
	switch len(ps) {
	case 0:
		return out, nil, nil
	case 1:
		x := ps[0]
		if await {
			if _, err := x.Await(); err != nil {
				return nil, nil, err
			}
			return out, nil, nil
		}
		return nil, promise.Then(x, func(any) []any { return out }), nil
	default:
		x := promise.All(ps...)
		if await {
			if _, err := x.Await(); err != nil {
				return nil, nil, err
			}
			return out, nil, nil
		}
		return nil, promise.Then(x, func([]any) []any { return out }), nil
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
	get := &currentHandlerInfoFactory{}
	handler.Handle(get, false, handler)
	return get.factory
}


// currentHandlerInfoFactory Resolves the current HandlerInfoFactory
type currentHandlerInfoFactory struct {
	factory HandlerInfoFactory
}

func (f *currentHandlerInfoFactory) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*Composition); ok {
		callback = comp.callback
	}
	if get, ok := callback.(*currentHandlerInfoFactory); ok {
		get.factory = f.factory
		return Handled
	}
	return NotHandled
}

func (f *currentHandlerInfoFactory) SuppressDispatch() {}

func (f *currentHandlerInfoFactory) CabBatch() bool {
	return false
}


var suppressDispatchType = internal.TypeOf[suppressDispatch]()
