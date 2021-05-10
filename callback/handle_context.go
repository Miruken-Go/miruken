package callback

import "reflect"

var (
	empty = new(emptyCtx)
)

// HandleContext

type HandleContext interface {
	Handler
	GetValue(key interface{}) interface{}
}

// emptyCtx

type emptyCtx int

func (c *emptyCtx) GetValue(
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

// withHandlerContext

type withHandlerContext struct {
	HandleContext
	handler Handler
}

func (c *withHandlerContext) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = &compositionScope{c}
	}
	return c.handler.Handle(callback, greedy, context).
		OtherwiseIf(greedy, func (HandleResult) HandleResult {
			return c.HandleContext.Handle(callback, greedy, context)
	})
}

// withHandlersContext

type withHandlersContext struct {
	HandleContext
	handlers []Handler
}

func (c *withHandlersContext) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = &compositionScope{c}
	}

	result := NotHandled

	for _, h := range c.handlers {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, context))
	}

	return result.OtherwiseIf(greedy, func (HandleResult) HandleResult {
		return c.HandleContext.Handle(callback, greedy, context)
	})
}

func AddHandlers(context HandleContext, handlers ... Handler) HandleContext {
	c := len(handlers)

	switch {
	case c == 1:
		return &withHandlerContext{context, handlers[0]}
	case c > 1:
		return &withHandlersContext{context, handlers}
	default:
		return context
	}
}

func RootHandler(handler Handler) HandleContext {
	if handler == nil {
		panic("cannot create root context from nil handler")
	}
	return &withHandlerContext{empty, handler}
}

// chainCtx

type chainCtx struct {
	primary   HandleContext
	secondary HandleContext
}

func (c *chainCtx) GetValue(key interface{}) interface{} {
	if value := c.primary.GetValue(key); value != nil {
		return value
	}
	return c.secondary.GetValue(key)
}

func (c *chainCtx) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = &compositionScope{c}
	}
	return c.primary.Handle(callback, greedy, context).
		OtherwiseIf(greedy, func(HandleResult) HandleResult {
			return c.secondary.Handle(callback, greedy, context)
		})
}

func ChainContexts(primary HandleContext, secondary HandleContext) HandleContext {
	return &chainCtx{primary, secondary}
}

// withKeyValueContext

type withKeyValueContext struct {
	HandleContext
	key, val interface{}
}

func (c *withKeyValueContext) GetValue(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.HandleContext.GetValue(key)
}

func WithKeyValue(parent HandleContext, key, val interface{}) HandleContext {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &withKeyValueContext{parent, key, val}
}
