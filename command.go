package miruken

import (
	"fmt"
	"reflect"
)

// Command handles messages Covariantly.
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
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
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
) (reset func (), approved bool) {
	if guard, ok := c.callback.(CallbackGuard); ok {
		return guard.CanDispatch(handler, binding)
	}
	return nil, true
}

func (c *Command) CanInfer() bool {
	if infer, ok := c.callback.(interface{CanInfer() bool}); ok {
		return infer.CanInfer()
	}
	return true
}

func (c *Command) CanFilter() bool {
	if infer, ok := c.callback.(interface{CanFilter() bool}); ok {
		return infer.CanFilter()
	}
	return true
}

func (c *Command) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(c.Policy(), handler, c.callback, c, greedy, composer, c).
		OtherwiseHandledIf(len(c.results) > count)
}

type CommandBuilder struct {
	CallbackBuilder
	callback interface{}
}

func (b *CommandBuilder) WithCallback(
	callback interface{},
) *CommandBuilder {
	b.callback = callback
	return b
}

func (b *CommandBuilder) NewCommand() *Command {
	return &Command{
		CallbackBase: b.Callback(),
		callback: b.callback,
	}
}

func Invoke(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetValue(target)
	command := new(CommandBuilder).
		WithCallback(callback).
		NewCommand()
	if result := handler.Handle(command, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return NotHandledError{callback}
	}
	command.CopyResult(tv)
	return nil
}

func InvokeAll(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	builder := new(CommandBuilder).WithCallback(callback)
	builder.WithMany()
	command := builder.NewCommand()
	if result := handler.Handle(command, true, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return NotHandledError{callback}
	}
	command.CopyResult(tv)
	return nil
}

// Handles policy for handling callbacks contravariantly.
type Handles struct {
	contravariantPolicy
}

func (h *Handles) Key(callback Callback) interface{} {
	if cmd, ok := callback.(*Command); ok {
		return reflect.TypeOf(cmd.Callback())
	}
	panic(fmt.Sprintf("Unrecognized Handles callback %#v", callback))
}

func HandlesPolicy() Policy { return _handles }

var _handles = RegisterPolicy(new(Handles))