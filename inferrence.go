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
	context := &HandleContext{callback, rawCallback, composer, results}
	return h.descriptor.Dispatch(policy, h, constraint, greedy, context)
}

// bindingIntercept intercepts Binding invocations to handler inference.
type bindingIntercept struct {
	handlerType reflect.Type
	Binding
}

func (b *bindingIntercept) Invoke(
	receiver  interface{},
	context  *HandleContext,
) ([]interface{}, error) {
	if ctor, ok := b.Binding.(*constructorBinding); ok {
		return ctor.Invoke(nil, context)
	}
	builder := new(ResolvingBuilder).WithCallback(context.RawCallback)
	builder.WithKey(b.handlerType)
	resolving := builder.NewResolving()
	if result := context.Composer.Handle(resolving, false, nil); result.IsError() {
		return nil, result.Error()
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
					pb.insert(&bindingIntercept{
						descriptor.handlerType,
						elem.Value.(Binding),
					})
					elem = elem.Next()
				}
				for _, bs := range pb.invar {
					for _, b := range bs {
						pb.insert(&bindingIntercept{
							descriptor.handlerType, b,
						})
					}
				}
			}
		}
	}
	return &inferenceHandler {
		&HandlerDescriptor{
			handlerType: _inferenceHandlerType,
			bindings:    bindings},
	}
}

var _inferenceHandlerType = reflect.TypeOf((*inferenceHandler)(nil)).Elem()