package miruken

import "reflect"

type (
	inferenceHandler struct {
		descriptor *HandlerDescriptor
	}

	inference struct {
		callback any
		greedy   bool
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
	policy      Policy,
	callback    any,
	rawCallback Callback,
	greedy      bool,
	composer    Handler,
) HandleResult {
	if test, ok := rawCallback.(interface{CanInfer() bool}); ok && !test.CanInfer() {
		return NotHandled
	}
	infer := &inference{callback: callback, greedy: greedy}
	return h.descriptor.Dispatch(policy, h, infer, rawCallback, greedy, composer)
}

func (h *inferenceHandler) SuppressDispatch() {}

// ctorIntercept intercepts constructor Binding invocations.
type ctorIntercept struct {
	*constructorBinding
}

func (b *ctorIntercept) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	context.binding = b.constructorBinding
	if infer, ok := context.Callback().(*inference); ok {
		context.callback = infer.callback
	}
	return b.constructorBinding.Invoke(context)
}

// funcIntercept intercepts function Binding invocations.
type funcIntercept struct {
	*funcBinding
}

func (b *funcIntercept) Invoke(
	context      HandleContext,
	explicitArgs ... any,
) ([]any, error) {
	context.binding = b.funcBinding
	if infer, ok := context.Callback().(*inference); ok {
		context.callback = infer.callback
	}
	return b.funcBinding.Invoke(context)
}

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
	var greedy bool
	handlerType := b.handlerType
	if infer, ok := context.Callback().(*inference); ok {
		greedy = infer.greedy
		context.callback = infer.callback
		if resolved := infer.resolved; resolved == nil {
			infer.resolved = map[reflect.Type]struct{} { handlerType: {} }
		} else if _, found := resolved[handlerType]; !found {
			resolved[handlerType] = struct{}{}
		} else {
			return nil, nil
		}
	}
	context.binding  = b.methodBinding
	rawCallback     := context.RawCallback()
	parent, _       := rawCallback.(*Provides)
	var builder ResolvingBuilder
	builder.
		WithCallback(rawCallback).
		WithGreedy(greedy).
		WithParent(parent).
		WithKey(handlerType)
	resolving := builder.NewResolving()
	if result := context.Composer().Handle(resolving, true, nil); result.IsError() {
		return nil, result.Error()
	} else {
		return []any{result}, nil
	}
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
					interceptBinding(elem.Value.(Binding), pb, handlerType,true)
				}
				// Only need the first of each invariant since it is
				// just to link the actual handler descriptor.
				for _, bs := range bs.invariant {
					if len(bs) > 0 {
						interceptBinding(bs[0], pb, handlerType, true)
					}
				}
				// Only need one unknown binding to create link.
				if last := bs.variant.Back(); last != nil {
					binding := last.Value.(Binding)
					if binding.Key() == _anyType {
						interceptBinding(binding, pb, handlerType, false)
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

func interceptBinding(
	binding         Binding,
	bindings        *policyBindings,
	handlerType     reflect.Type,
	addConstructor  bool,
) {
	switch b := binding.(type) {
	case *constructorBinding:
		if addConstructor {
			bindings.insert(&ctorIntercept{b})
		}
	case *methodBinding:
		bindings.insert(&methodIntercept{b, handlerType})
	case *funcBinding:
		bindings.insert(&funcIntercept{b})
	}
}

var _inferenceHandlerType = TypeOf[*inferenceHandler]()