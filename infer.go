package miruken

import (
	"reflect"
)

type (
	// inferenceHandler aggregates the bindings of a set of
	// handlers to provide a central point of interception
	// for inference capability.
	inferenceHandler struct {
		descriptor *HandlerDescriptor
	}

	// inferenceGuard prevents the same handler from being
	// called too many times during a dispatch.  This can
	// occur when a handler matches multiple bindings for
	// the same callback (covariance and contravariance).
	//
	// e.g.
	// type ListProvider struct{}
	//
	// func (f *ListProvider) ProvideFooSlice(*miruken.Provides) []*Foo {
	//	  return []*Foo{{Counted{1}}, {Counted{2}}}
	// }
	//
	//  func (f *ListProvider) ProvideFooArray(*miruken.Provides) [2]*Bar {
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
	return h.descriptor.Dispatch(policy, h, callback, greedy, composer, &inferenceGuard{})
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
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	handlerType := b.handlerType
	callback    := context.Callback()
	parent, _   := callback.(*Provides)
	var builder ResolvingBuilder
	builder.
		WithCallback(callback).
		WithGreedy(context.Greedy()).
		WithParent(parent).
		WithKey(handlerType)
	resolving := builder.NewResolving()
	if result := context.Composer().Handle(resolving, true, nil); result.IsError() {
		return nil, result.Error()
	} else {
		return []any{result}, nil
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
	factory HandlerDescriptorFactory,
	specs   []HandlerSpec,
) *inferenceHandler {
	if factory == nil {
		panic("factory cannot be nil")
	}
	bindings := make(policyBindingsMap)
	for _, spec := range specs {
		if descriptor, added, err := factory.RegisterHandler(spec); err != nil {
			panic(err)
		} else if added {
			var handlerType reflect.Type
			if h, ok := descriptor.spec.(HandlerTypeSpec); ok {
				handlerType = h.Type()
			}
			for policy, bs := range descriptor.bindings {
				pb := bindings.forPolicy(policy)
				// Us bs.index vs.variant since inference ONLY needs a
				// single binding to infer the handler type for a
				// specific key.
				for _, elem := range bs.index {
					linkBinding(elem.Value.(Binding), pb, handlerType,true)
				}
				// Only need the first of each invariant since it is
				// just to link the actual handler descriptor.
				for _, bs := range bs.invariant {
					if len(bs) > 0 {
						linkBinding(bs[0], pb, handlerType, true)
					}
				}
				// Only need one unknown binding to create link.
				if last := bs.variant.Back(); last != nil {
					binding := last.Value.(Binding)
					if binding.Key() == _anyType {
						linkBinding(binding, pb, handlerType, false)
					}
				}
			}
		}
	}
	return &inferenceHandler {
		&HandlerDescriptor{
			spec:     HandlerTypeSpec{_inferenceHandlerType},
			bindings: bindings,
		},
	}
}

func linkBinding(
	binding         Binding,
	bindings        *policyBindings,
	handlerType     reflect.Type,
	addConstructor  bool,
) {
	switch b := binding.(type) {
	case *constructorBinding:
		if addConstructor {
			bindings.insert(b)
		}
	case *methodBinding:
		bindings.insert(&methodIntercept{b, handlerType})
	case *funcBinding:
		bindings.insert(b)
	}
}

var _inferenceHandlerType = TypeOf[*inferenceHandler]()