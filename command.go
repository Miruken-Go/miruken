package miruken

type Command struct {
	CallbackBase
	callback interface{}
}

func (c *Command) Callback() interface{} {
	return c.callback
}

func (c *Command) Policy() Policy {
	return HandlesPolicy()
}

func (c *Command) ReceiveResult(
	result interface{},
	strict bool,
	greedy bool,
	ctx    HandleContext,
) (accepted bool) {
	if result == nil {
		return false
	}
	c.results = append(c.results, result)
	c.result  = nil
	return true
}

func (c *Command) CanDispatch(
	handler     interface{},
	binding     Binding,
) (reset func (interface{}), approved bool) {
	if guard, ok := c.callback.(CallbackGuard); ok {
		return guard.CanDispatch(handler, binding)
	}
	return nil, true
}

func (c *Command) Dispatch(
	handler interface{},
	greedy  bool,
	ctx     HandleContext,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(c.Policy(), handler, c.callback, c, nil, greedy, ctx, c).
		OtherwiseHandledIf(len(c.results) > count)
}

func NewCommand(callback interface{}, many bool) *Command {
	var command = new(Command)
	command.callback = callback
	command.many     = many
	return command
}