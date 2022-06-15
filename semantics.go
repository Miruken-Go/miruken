package miruken

type SemanticFlags uint8

const (
	SemanticNone      SemanticFlags = 0
	SemanticBroadcast SemanticFlags = 1 << iota
	SemanticBestEffort
	SemanticNotify  = SemanticBroadcast | SemanticBestEffort
)

// CallbackSemantics captures semantic options
type CallbackSemantics struct {
	Composition
	options   SemanticFlags
	specified SemanticFlags
}

func (c *CallbackSemantics) CanInfer() bool {
	return false
}

func (c *CallbackSemantics) CanFilter() bool {
	return false
}

func (c *CallbackSemantics) HasOption(options SemanticFlags) bool  {
	return (c.options & options) == options
}

func (c *CallbackSemantics) SetOption(options SemanticFlags, enabled bool) {
	if enabled {
		c.options = c.options | options
	} else {
		c.options = c.options & ^options
	}
	c.specified = c.specified | options
}

func (c *CallbackSemantics) IsSpecified(options SemanticFlags) bool  {
	return (c.specified & options) == options
}

func (c *CallbackSemantics) MergeInto(semantics *CallbackSemantics) {
	c.mergeOption(semantics, SemanticBestEffort)
	c.mergeOption(semantics, SemanticBroadcast)
}

func (c *CallbackSemantics) mergeOption(
	semantics *CallbackSemantics,
	option SemanticFlags,
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
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, c)
	if comp, ok := callback.(*Composition); ok {
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
	if c.semantics.IsSpecified(SemanticBroadcast) {
		greedy = c.semantics.HasOption(SemanticBroadcast)
	}
	if c.semantics.IsSpecified(SemanticBestEffort) &&
		c.semantics.HasOption(SemanticBestEffort) {
		if result := c.Handler.Handle(callback, greedy, composer); result.IsError() {
			switch result.Error().(type) {
			case NotHandledError: return Handled
			case RejectedError: return Handled
			default: return result
			}
		} else {
			return Handled
		}
	}
	return c.Handler.Handle(callback, greedy, composer)
}

func GetSemantics(handler Handler) *CallbackSemantics {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	semantics := &CallbackSemantics{}
	if result := handler.Handle(semantics, true, handler); result.IsHandled() {
		return semantics
	}
	return nil
}

func CallWith(semantics SemanticFlags) Builder {
	options := CallbackSemantics{
		options:   semantics,
		specified: semantics,
	}
	return BuilderFunc(func (handler Handler) Handler {
		return &callSemantics{handler, options}
	})
}

var (
	Broadcast  = CallWith(SemanticBroadcast)
	BestEffort = CallWith(SemanticBestEffort)
	Notify     = CallWith(SemanticNotify)
)