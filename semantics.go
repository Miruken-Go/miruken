package miruken

type semanticFlags uint8

const (
	semanticNone semanticFlags = 0
	semanticDuck = 1 << iota
	semanticStrict
	semanticBroadcast
	semanticBestEffort
	semanticNotify = semanticBroadcast | semanticBestEffort
)

// CallbackSemantics captures semantic options
type CallbackSemantics struct {
	Composition
	options semanticFlags
	specified semanticFlags
}

func (c *CallbackSemantics) CanInfer() bool {
	return false
}

func (c *CallbackSemantics) HasOption(options semanticFlags) bool  {
	return (c.options & options) == options
}

func (c *CallbackSemantics) SetOption(options semanticFlags, enabled bool) {
	if enabled {
		c.options = c.options | options
	} else {
		c.options = c.options & ^options
	}
	c.specified = c.specified | options
}

func (c *CallbackSemantics) IsSpecified(options semanticFlags) bool  {
	return (c.specified & options) == options
}

func (c *CallbackSemantics) MergeInto(semantics *CallbackSemantics) {
	c.mergeOption(semantics, semanticDuck)
	c.mergeOption(semantics, semanticStrict)
	c.mergeOption(semantics, semanticBestEffort)
	c.mergeOption(semantics, semanticBroadcast)
}

func (c *CallbackSemantics) mergeOption(
	semantics *CallbackSemantics,
	option     semanticFlags,
) {
	if c.IsSpecified(option) && !semantics.IsSpecified(option) {
		semantics.SetOption(option, c.HasOption(option))
	}
}

// callSemantics applies CallbackSemantics.
type callSemantics struct {
	Handler
	semantics CallbackSemantics
}

func (c *callSemantics) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	if comp, ok := callback.(Composition); ok {
		if _, yes := comp.Callback().(CallbackSemantics); yes {
			return NotHandled
		}
	}
	switch cb := callback.(type) {
	case *CallbackSemantics:
		c.semantics.MergeInto(cb)
		if greedy {
			c.Handler.Handle(callback, greedy, composer)
		}
		return Handled
	case Composition:
		return c.Handler.Handle(callback, greedy, composer)
	}
	if c.semantics.IsSpecified(semanticBroadcast) {
		greedy = c.semantics.HasOption(semanticBroadcast)
	}
	if c.semantics.IsSpecified(semanticBestEffort) &&
		c.semantics.HasOption(semanticBestEffort) {
		if result := c.Handler.Handle(callback, greedy, composer); result.IsError() {
			switch result.Error().(type) {
			case *NotHandledError: return Handled
			case *RejectedError: return Handled
			default: return result
			}
		} else {
			return Handled
		}
	}
	return c.Handler.Handle(callback, greedy, composer)
}

func GetSemantics(handler Handler) *CallbackSemantics {
	if handler == nil {
		panic("handler cannot be nil")
	}
	semantics := &CallbackSemantics{}
	if result := handler.Handle(semantics, true, handler); result.IsHandled() {
		return semantics
	}
	return nil
}

func WithCallSemantics(semantics semanticFlags) Builder {
	return BuilderFunc(func (handler Handler) Handler {
		return &callSemantics{handler,
			CallbackSemantics{
				options:   semantics,
				specified: semantics}}
	})
}

var (
	WithDuckTyping Builder = WithCallSemantics(semanticDuck)
	WithStrict     Builder = WithCallSemantics(semanticStrict)
	WithBroadcast  Builder = WithCallSemantics(semanticBroadcast)
	WithBestEffort Builder = WithCallSemantics(semanticBestEffort)
	WithNotify     Builder = WithCallSemantics(semanticNotify)
)