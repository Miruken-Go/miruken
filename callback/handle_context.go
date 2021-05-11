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

func NewHandleContext(handlers ... interface{}) HandleContext {
	return AddHandlers(empty, handlers...)
}

func AddHandlers(parent HandleContext, handlers ... interface{}) HandleContext {
	if parent == nil {
		panic("cannot add handlers to a nil parent")
	}

	hs := normalizeHandlers(handlers)

	switch c := len(hs); {
	case c == 1:
		return &withHandlerContext{parent, hs[0]}
	case c > 1:
		return &withHandlersContext{parent, hs}
	default:
		return parent
	}
}

func normalizeHandlers(handlers []interface{}) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}

// chainCtx

type chainCtx struct {
	contexts []HandleContext
}

func (c *chainCtx) GetValue(key interface{}) interface{} {
	for _, ctx := range c.contexts {
		if value := ctx.GetValue(key); value != nil {
			return value
		}
	}
	return nil
}

func (c *chainCtx) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = &compositionScope{c}
	}

	result := NotHandled

	for _, ctx := range c.contexts {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(ctx.Handle(callback, greedy, context))
	}

	return result
}

func Chain(contexts ... HandleContext) HandleContext {
	return &chainCtx{contexts}
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

func (c *withKeyValueContext) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if context == nil {
		context = &compositionScope{c}
	}
	return c.HandleContext.Handle(callback, greedy, context)
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
