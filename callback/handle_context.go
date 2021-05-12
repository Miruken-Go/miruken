package callback

import "reflect"

var (
	empty = new(emptyCtx)
)

// HandleContext

type HandleContext interface {
	Handler
	Value(key interface{}) interface{}
}

// emptyCtx

type emptyCtx int

func (c *emptyCtx) Value(
	interface{},
) interface{} {
	return nil
}

func (c *emptyCtx) Handle(
	interface{},
	bool,
	HandleContext,
) HandleResult {
	return NotHandled
}

// withKeyValueCtx

type withKeyValueCtx struct {
	HandleContext
	key, val interface{}
}

func (c *withKeyValueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.HandleContext.Value(key)
}

func (c *withKeyValueCtx) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	TryInitializeContext(&ctx, c)
	return c.HandleContext.Handle(callback, greedy, ctx)
}

func WithKeyValue(
	parent HandleContext,
	key, val interface{},
) HandleContext {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &withKeyValueCtx{parent, key, val}
}

// withHandlerCtx

type withHandlerCtx struct {
	HandleContext
	handler Handler
}

func (c *withHandlerCtx) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	TryInitializeContext(&ctx, c)
	return c.handler.Handle(callback, greedy, ctx).
		OtherwiseIf(greedy, func (HandleResult) HandleResult {
			return c.HandleContext.Handle(callback, greedy, ctx)
	})
}

// withHandlersCtx

type withHandlersCtx struct {
	HandleContext
	handlers []Handler
}

func (c *withHandlersCtx) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	TryInitializeContext(&ctx, c)

	result := NotHandled

	for _, h := range c.handlers {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, ctx))
	}

	return result.OtherwiseIf(greedy, func (HandleResult) HandleResult {
		return c.HandleContext.Handle(callback, greedy, ctx)
	})
}

func AddHandlers(
	parent HandleContext,
	handlers ... interface{},
) HandleContext {
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
		return &withHandlerCtx{parent, hs[0]}
	case c > 1:
		return &withHandlersCtx{parent, hs}
	default:
		return parent
	}
}

// chainCtx

type chainCtx struct {
	contexts []HandleContext
}

func (c *chainCtx) Value(key interface{}) interface{} {
	for _, ctx := range c.contexts {
		if value := ctx.Value(key); value != nil {
			return value
		}
	}
	return nil
}

func (c *chainCtx) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	TryInitializeContext(&ctx, c)

	result := NotHandled

	for _, ctx := range c.contexts {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(ctx.Handle(callback, greedy, ctx))
	}

	return result
}

func Chain(contexts ... HandleContext) HandleContext {
	return &chainCtx{contexts}
}

func TryInitializeContext(
	incoming *HandleContext,
	receiver  HandleContext,
) {
	if *incoming == nil {
		*incoming = &compositionScope{receiver}
	}
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

var factoryKey key

func WithHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) HandleContextOption {
	if factory == nil {
		panic("nil factory")
	}
	return handleContextOptionFunc(func (opts *handleContextOptions) {
		opts.factory = factory
	})
}

func GetHandlerDescriptorFactory(
	ctx HandleContext,
) HandlerDescriptorFactory {
	switch f := ctx.Value(&factoryKey).(type) {
	case HandlerDescriptorFactory: return f
	default: return nil
	}
}

func normalizeHandlers(handlers []interface{}) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}

func NewHandleContext(
	opts ... HandleContextOption,
) HandleContext {
	var options handleContextOptions
	for _, o := range opts {
		o.applyHandleContextOption(&options)
	}

	var ctx HandleContext = empty

	factory := options.factory
	if factory == nil {
		factory = &mutableFactory{}
	}
	ctx = WithKeyValue(ctx, &factoryKey, factory)

	if len(options.handlers) > 0 {
		ctx = AddHandlers(ctx, options.handlers...)
	}

	return ctx
}
