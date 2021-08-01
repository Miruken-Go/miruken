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
	rawCallback interface{},
	constraint  interface{},
	greedy      bool,
	composer    Handler,
	results     ResultReceiver,
) HandleResult {
	if infer, ok := rawCallback.(interface{CanInfer() bool}); ok && !infer.CanInfer() {
		return NotHandled
	}
	context := HandleContext{callback, rawCallback, composer, results}
	return h.descriptor.Dispatch(policy, h, constraint, greedy, context)
}

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
	builder := new(ResolvingBuilder).WithCallback(context.RawCallback)
	builder.WithKey(b.handlerType)
	resolving := builder.NewResolving()
	if result := context.Composer.Handle(resolving, false, nil); result.IsError() {
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
		if typ.AssignableTo(_suppressType) {
			continue
		}
		if descriptor, added, err := factory.RegisterHandlerType(typ); err != nil {
			panic(err)
		} else if added {
			for policy, bs := range descriptor.bindings {
				pb   := bindings.getBindings(policy)
				elem := bs.typed.Front()
				for elem != nil {
					binding := elem.Value.(Binding)
					_, ctorBinding := binding.(*constructorBinding)
					pb.insert(&bindingIntercept{
						descriptor.handlerType,
						!ctorBinding,
						binding,
					})
					elem = elem.Next()
				}
				for _, bs := range pb.invar {
					for _, b := range bs {
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