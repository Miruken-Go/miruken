package miruken

import "reflect"

// Builder augments a Handler.
type Builder interface {
	Build(Handler) Handler
}

type BuilderFunc func(Handler) Handler

func (f BuilderFunc) Build(
	handler Handler,
) Handler { return f(handler) }

var nullBuilder BuilderFunc = func(handler Handler) Handler {
	return handler
}

func composeBuilder2(builder1 Builder, builder2 Builder) Builder {
	return BuilderFunc(func(handler Handler) Handler {
		if builder2 != nil {
			handler = builder2.Build(handler)
		}
		if builder1 != nil {
			handler = builder1.Build(handler)
		}
		return handler
	})
}

func ComposeBuilders(builders ... Builder) Builder {
	switch len(builders) {
	case 0: return nullBuilder
	case 1: return builders[0]
	default:
		builder := builders[0]
		for _, b := range builders[1:] {
			builder = composeBuilder2(builder, b)
		}
		return builder
	}
}

func PipeBuilders(builders ... Builder) Builder {
	switch len(builders) {
	case 0: return nullBuilder
	case 1: return builders[0]
	default:
		builder := builders[len(builders)-1]
		for i := len(builders)-2; i >= 0; i-- {
			builder = composeBuilder2(builder, builders[i])
		}
		return builder
	}
}

func Build(handler Handler, builders ... Builder) Handler {
	for _, b := range builders {
		if b != nil {
			handler = b.Build(handler)
		}
	}
	return handler
}

func AddHandlers(
	parent Handler,
	handlers ... interface{},
) Handler {
	if parent == nil {
		panic("cannot add handlers to a nil parent")
	}

	var factory HandlerDescriptorFactory

	for _, handler := range handlers {
		if _, ok := handler.(SuppressDispatch); ok {
			continue
		}
		if factory == nil {
			if factory = GetHandlerDescriptorFactory(parent); factory == nil {
				break
			}
		}
		typ := reflect.TypeOf(handler)
		if _, _, err := factory.RegisterHandlerType(typ); err != nil {
			panic(err)
		}
	}

	hs := normalizeHandlers(handlers)

	switch c := len(hs); {
	case c == 1:
		return &withHandler{parent, hs[0]}
	case c > 1:
		return &withHandlers{parent, hs}
	default:
		return parent
	}
}

func With(values ... interface{}) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		var valueHandlers []interface{}
		for _, val := range values {
			if val != nil {
				valueHandlers = append(valueHandlers, NewProvider(val))
			}
		}
		if len(valueHandlers) > 0 {
			return AddHandlers(handler, valueHandlers...)
		}
		return handler
	})
}

// withHandler composes two Handlers.
type withHandler struct {
	Handler
	handler Handler
}

func (w *withHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, w)
	return w.handler.Handle(callback, greedy, composer).
		OtherwiseIf(greedy, func (HandleResult) HandleResult {
			return w.Handler.Handle(callback, greedy, composer)
		})
}

func (w *withHandler) suppressDispatch() {}

// withHandlers composes any number of Handlers.
type withHandlers struct {
	Handler
	handlers []Handler
}

func (w *withHandlers) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, w)

	result := NotHandled

	for _, h := range w.handlers {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, composer))
	}
	return result.OtherwiseIf(greedy, func (HandleResult) HandleResult {
		return w.Handler.Handle(callback, greedy, composer)
	})
}

func (w *withHandlers) suppressDispatch() {}

func WithHandlers(handlers ... interface{}) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		return AddHandlers(handler, handlers...)
	})
}

func WithHandlerTypes(types ... reflect.Type) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		if factory := GetHandlerDescriptorFactory(handler); factory != nil {
			return &withHandler{handler, newInferenceHandler(factory, types)}
		} else {
			panic("unable to obtain the HandlerDescriptorFactory")
		}
	})
}

type FilterFunc = func(
	callback interface{},
	composer Handler,
	proceed  func() HandleResult,
) HandleResult

// filterHandler applies a filter to a Handler.
type filterHandler struct {
	Handler
	filter    FilterFunc
	reentrant bool
}

func (f *filterHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, f)

	if !f.reentrant {
		if _, ok := callback.(*Composition); ok {
			return f.Handler.Handle(callback, greedy, composer)
		}
	}
	return f.filter(callback, composer, func() HandleResult {
		return f.Handler.Handle(callback, greedy, composer)
	})
}

func WithFilter(filter FilterFunc, reentrant bool) Builder {
	if filter == nil {
		panic("filter cannot be nil")
	}
	return BuilderFunc(func (handler Handler) Handler {
		return &filterHandler{handler, filter, reentrant}
	})
}


func tryInitializeComposer(
	incoming *Handler,
	receiver  Handler,
) {
	if *incoming == nil {
		*incoming = &compositionScope{receiver}
	}
}

func NewRootHandler(builders ... Builder) Handler {
	factory := NewMutableHandlerDescriptorFactory()
	var handler Handler = &getHandlerDescriptorFactory{factory}
	return Build(handler, builders...)
}

func normalizeHandlers(handlers []interface{}) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}
