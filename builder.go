package miruken

import "reflect"

type Builder interface {
	Build(Handler) Handler
}

type BuilderFunc func(Handler) Handler

func (f BuilderFunc) Build(
	handler Handler,
) Handler { return f(handler) }

func Build(handler Handler, builders ... Builder) Handler {
	for _, b := range builders {
		if b != nil {
			handler = b.Build(handler)
		}
		if handler == nil {
			panic("handler cannot be nil")
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

	if factory := GetHandlerDescriptorFactory(parent); factory != nil {
		for _, handler := range handlers {
			typ := reflect.TypeOf(handler)
			if _, err := factory.RegisterHandlerType(typ); err != nil {
				panic(err)
			}
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


// withHandler

type withHandler struct {
	Handler
	handler Handler
}

func (c *withHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, c)
	return c.handler.Handle(callback, greedy, composer).
		OtherwiseIf(greedy, func (HandleResult) HandleResult {
			return c.Handler.Handle(callback, greedy, composer)
		})
}

// withHandlers

type withHandlers struct {
	Handler
	handlers []Handler
}

func (c *withHandlers) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, c)

	result := NotHandled

	for _, h := range c.handlers {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, composer))
	}

	return result.OtherwiseIf(greedy, func (HandleResult) HandleResult {
		return c.Handler.Handle(callback, greedy, composer)
	})
}

func WithHandlers(handlers ... interface{}) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		return AddHandlers(handler, handlers...)
	})
}

func WithHandlerTypes(types ... reflect.Type) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		if factory := GetHandlerDescriptorFactory(handler); factory != nil {
			for _, typ := range types {
				if _, err := factory.RegisterHandlerType(typ); err != nil {
					panic(err)
				}
			}
		}
		return &withHandler{handler, NewInferenceHandler(types...)}
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

func NewRootHandler() Handler {
	factory := NewMutableHandlerDescriptorFactory()
	return &getHandlerDescriptorFactory{factory}
}

func normalizeHandlers(handlers []interface{}) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}
