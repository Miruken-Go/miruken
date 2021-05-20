package miruken

import (
	"fmt"
	"reflect"
)

// Handler

type Handler interface {
	Handle(
		callback interface{},
		greedy   bool,
		composer Handler,
	) HandleResult
}

// handlerAdapter

type handlerAdapter struct {
	handler interface{}
}

func (h *handlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h.handler, callback, greedy, composer)
}

// NotHandledError

type NotHandledError struct {
	callback interface{}
}

func (e *NotHandledError) Error() string {
	return fmt.Sprintf("callback %#v not handled", e.callback)
}

func DispatchCallback(
	handler  interface{},
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if handler == nil {
		return NotHandled
	}
	if dispatch, ok := callback.(CallbackDispatcher); ok {
		return dispatch.Dispatch(handler, greedy, composer)
	}
	command := NewCommand(callback,false)
	return command.Dispatch(handler, greedy, composer)
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

func AddHandlers(
	parent Handler,
	handlers ... interface{},
) Handler {
	if parent == nil {
		panic("cannot add handlers to a nil parent")
	}

	if factory := GetHandlerDescriptorFactory(parent); factory != nil {
		for _, h := range handlers {
			handlerType := reflect.TypeOf(h)
			if _, err := factory.RegisterHandlerType(handlerType); err != nil {
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

func NewHandleContext(
	opts ...HandleContextOption,
) Handler {
	var options handleContextOptions
	for _, o := range opts {
		o.applyHandleContextOption(&options)
	}

	factory := options.factory
	if factory == nil {
		factory = NewMutableHandlerDescriptorFactory()
	}

	var root Handler = &getHandlerDescriptorFactory{factory}
	if len(options.handlers) > 0 {
		root = AddHandlers(root, options.handlers...)
	}
	return root
}

// getHandlerDescriptorFactory is a special Handler to resolve the
// current HandlerDescriptorFactory

type getHandlerDescriptorFactory struct {
	factory HandlerDescriptorFactory
}

func (g *getHandlerDescriptorFactory) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if comp, ok := callback.(*composition); ok {
		callback = comp.callback
	}
	if getFactory, ok := callback.(*getHandlerDescriptorFactory); ok {
		getFactory.factory = g.factory
		return Handled
	}
	return NotHandled
}

func GetHandlerDescriptorFactory(
	handler Handler,
) HandlerDescriptorFactory {
	get := &getHandlerDescriptorFactory{}
	handler.Handle(get, false, handler)
	return get.factory
}

type handleContextOptions struct {
	handlers []interface{}
	factory  HandlerDescriptorFactory
}

type HandleContextOption interface {
	applyHandleContextOption(*handleContextOptions)
}

type handleContextOptionFunc func(*handleContextOptions)

func (f handleContextOptionFunc) applyHandleContextOption(
	opts *handleContextOptions,
) { f(opts) }

func WithHandlers(handlers ... interface{}) HandleContextOption {
	return handleContextOptionFunc(func (opts *handleContextOptions) {
		opts.handlers = append(opts.handlers, handlers...)
	})
}

func WithHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) HandleContextOption {
	if factory == nil {
		panic("factory cannot be nil")
	}
	return handleContextOptionFunc(func (opts *handleContextOptions) {
		opts.factory = factory
	})
}

func normalizeHandlers(handlers []interface{}) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}

func tryInitializeComposer(
	incoming *Handler,
	receiver  Handler,
) {
	if *incoming == nil {
		*incoming = &compositionScope{receiver}
	}
}

var (
	_handlerType = reflect.TypeOf((*Handler)(nil)).Elem()
)