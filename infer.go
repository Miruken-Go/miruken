package miruken

import (
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// inferenceHandler aggregates the bindings of a set of
	// handlers to provide a central point of interception
	// for inference capability.
	inferenceHandler struct {
		info *HandlerInfo
	}

	// inferenceGuard prevents the same handler from being
	// called too many times during a dispatch.  This can
	// occur when a handler matches multiple bindings for
	// the same callback (covariance and contravariance).
	//
	// e.g.
	// type ListProvider struct{}
	//
	// func (f *ListProvider) ProvideFooSlice(*provides.Build) []*Foo {
	//	  return []*Foo{{Counted{1}}, {Counted{2}}}
	// }
	//
	//  func (f *ListProvider) ProvideFooArray(*provides.Build) [2]*Bar {
	//	  return [2]*Bar{{Counted{3}}, {Counted{4}}}
	// }
	//
	// would result in ListProvider being called too many times and
	// resulting in additional resolved Counted instances.
	//
	// type (
	//    Counted struct { count int }
	//    Foo     struct { Counted }
	//    Bar     struct { Counted }
	// )
	//
	// var counted []Counter
	// miruken.ResolveAll(handler, &counted)
	//
	inferenceGuard struct {
		resolved map[reflect.Type]struct{}
	}
)

func (h *inferenceHandler) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h, callback, greedy, composer)
}

func (h *inferenceHandler) DispatchPolicy(
	policy   Policy,
	callback Callback,
	greedy   bool,
	composer Handler,
) HandleResult {
	if test, ok := callback.(interface{CanInfer() bool}); ok && !test.CanInfer() {
		return NotHandled
	}
	return h.info.Dispatch(policy, h, callback, greedy, composer, &inferenceGuard{})
}

func (h *inferenceHandler) SuppressDispatch() {}

// methodIntercept intercepts method Binding invocations.
type methodIntercept struct {
	*methodBinding
	handlerType reflect.Type
}

func (b *methodIntercept) Filters() []FilterProvider {
	return nil
}

func (b *methodIntercept) SkipFilters() bool {
	return true
}

func (b *methodIntercept) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	handlerType := b.handlerType
	callback    := ctx.Callback
	composer    := ctx.Composer
	parent, _   := callback.(*Provides)
	var builder ResolvesBuilder
	builder.
		WithCallback(callback).
		WithGreedy(ctx.Greedy).
		WithParent(parent).
		WithKey(handlerType)
	resolves := builder.New()
	if result := composer.Handle(resolves, true, nil); result.IsError() {
		return nil, nil, result.Error()
	} else if _, p := resolves.Result(false); p != nil {
		return nil, promise.Then(p, func(res any) []any {
			if !resolves.Succeeded() {
				panic(&NotHandledError{callback})
			}
			// Since this promise will be added to the actual callback's
			// results, return nil to ensure it is filtered out during a
			// call to Callback.Result().
			return nil
		}), nil
	} else {
		// Make the HandleResult the effective return to no
		// additional results are added to the actual callback.
		return []any{result}, nil, nil
	}
}

// CanDispatch is needed to prevent more than one method binding
// for the same handler from being inferred since only one is
// needed to initiate a ResolveAll for that handler type to
// dispatch the callback to all matching handlers.  Otherwise,
// multiple ResolveAll's will occur and the callback will be
// dispatched too many times to the same handlers.
func (g *inferenceGuard) CanDispatch(
	handler any,
	binding Binding,
) (reset func (), approved bool) {
	if methodBinding, ok := binding.(*methodIntercept); ok {
		handlerType := methodBinding.handlerType
		if resolved := g.resolved; resolved == nil {
			g.resolved = map[reflect.Type]struct{} { handlerType: {} }
		} else if _, found := resolved[handlerType]; !found {
			resolved[handlerType] = struct{}{}
		} else {
			return nil, false
		}
	}
	return nil, true
}

func newInferenceHandler(
	factory HandlerInfoFactory,
	specs   []HandlerSpec,
) *inferenceHandler {
	if factory == nil {
		panic("factory cannot be nil")
	}
	bindings := make(policyInfoMap)
	for _, spec := range specs {
		if info, added, err := factory.Register(spec); err != nil {
			panic(err)
		} else if added {
			var handlerType reflect.Type
			if h, ok := info.spec.(TypeSpec); ok {
				handlerType = h.Type()
			}
			for policy, bs := range info.bindings {
				pb := bindings.forPolicy(policy)
				// Us bs.index vs.variant since inference ONLY needs a
				// single binding to infer the handler type for a
				// specific key.
				for _, elem := range bs.index {
					linkBinding(policy, elem.Value.(Binding), pb, handlerType,true)
				}
				// Only need the first of each invariant since it is
				// just to link the actual handler info.
				for _, bs := range bs.invariant {
					if len(bs) > 0 {
						linkBinding(policy, bs[0], pb, handlerType, true)
					}
				}
				// Only need one unknown binding to create link.
				if last := bs.variant.Back(); last != nil {
					binding := last.Value.(Binding)
					if bt, ok := binding.Key().(reflect.Type); ok && anyType.AssignableTo(bt) {
						linkBinding(policy, binding, pb, handlerType, false)
					}
				}
			}
		}
	}
	return &inferenceHandler {
		&HandlerInfo{
			spec:     TypeSpec{inferenceHandlerType},
			bindings: bindings,
		},
	}
}

func linkBinding(
	policy          Policy,
	binding         Binding,
	bindings        *policyInfo,
	handlerType     reflect.Type,
	addConstructor  bool,
) {
	switch b := binding.(type) {
	case *constructorBinding:
		if addConstructor {
			bindings.insert(policy, b)
		}
	case *methodBinding:
		bindings.insert(policy, &methodIntercept{b, handlerType})
	case *funcBinding:
		bindings.insert(policy, b)
	}
}

var inferenceHandlerType = internal.TypeOf[*inferenceHandler]()