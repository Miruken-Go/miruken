package miruken

import "reflect"

type inferenceHandler struct {
	descriptor *HandlerDescriptor
}

func (h *inferenceHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h, callback, greedy, composer)
}

func (h *inferenceHandler) DispatchPolicy(
	policy      Policy,
	callback    interface{},
	rawCallback Callback,
	greedy      bool,
	composer    Handler,
	results     ResultReceiver,
) HandleResult {
	if infer, ok := rawCallback.(interface{CanInfer() bool}); ok && !infer.CanInfer() {
		return NotHandled
	}
	return h.descriptor.Dispatch(policy, h, callback, rawCallback, greedy, composer, results)
}

func (h *inferenceHandler) suppressDispatch() {}

// bindingIntercept intercepts Binding invocations to handler inference.
type bindingIntercept struct {
	handlerType reflect.Type
	skipFilters bool
	Binding
}

func (b *bindingIntercept) Filters() []FilterProvider {
	if b.skipFilters {
		return nil
	}
	return b.Binding.Filters()
}

func (b *bindingIntercept) SkipFilters() bool {
	return b.skipFilters
}

func (b *bindingIntercept) Invoke(
	context      HandleContext,
	explicitArgs ... interface{},
) ([]interface{}, error) {
	if ctor, ok := b.Binding.(*constructorBinding); ok {
		return ctor.Invoke(context)
	}
	builder := new(ResolvingBuilder).WithCallback(context.RawCallback())
	builder.WithKey(b.handlerType)
	resolving := builder.NewResolving()
	if result := context.Composer().Handle(resolving, false, nil); result.IsError() {
		return nil, result.Error()
	} else if !result.handled {
		return []interface{}{result}, nil
	}
	return nil, nil
}

func newInferenceHandler(
	factory HandlerDescriptorFactory,
	types   []reflect.Type,
) *inferenceHandler {
	if factory == nil {
		panic("factory cannot be nil")
	}
	bindings := make(policyBindingsMap)
	for _, typ := range types {
		if descriptor, added, err := factory.RegisterHandlerType(typ); err != nil {
			panic(err)
		} else if added {
			for policy, bs := range descriptor.bindings {
				pb := bindings.getBindings(policy)
				// Us bs.index vs.typed since inference ONLY needs a
				// single binding to infer the handler type for a
				// specific key.
				for _, elem := range bs.index {
					binding := elem.Value.(Binding)
					_, ctorBinding := binding.(*constructorBinding)
					pb.insert(&bindingIntercept{
						descriptor.handlerType,
						!ctorBinding,
						binding,
					})
				}
				for _, bs := range pb.invar {
					if len(bs) > 0 {
						b := bs[0]  // only need first
						_, ctorBinding := b.(*constructorBinding)
						pb.insert(&bindingIntercept{
							descriptor.handlerType, !ctorBinding, b,
						})
					}
				}
			}
		}
	}
	return &inferenceHandler {
		&HandlerDescriptor{
			handlerType: _inferenceHandlerType,
			bindings:    bindings,
		},
	}
}

var _inferenceHandlerType = reflect.TypeOf((*inferenceHandler)(nil)).Elem()