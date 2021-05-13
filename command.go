package miruken

import (
	"reflect"
)

type Command struct {
	callback interface{}
	many     bool
	results  []interface{}
	result   interface{}
}

func (c *Command) Many() bool {
	return c.many
}

func (c *Command) Callback() interface{} {
	return c.callback
}

func (c *Command) Policy() Policy {
	return HandlesPolicy()
}

func (c *Command) ResultType() reflect.Type {
	return nil
}

func (c *Command) Result() interface{} {
	if result := c.result; result == nil {
		if c.many {
			c.result = c.results
		} else {
			if len(c.results) == 0 {
				c.result = nil
			} else {
				c.result = c.results[0]
			}
		}
	}
	return c.result
}

func (c *Command) SetResult(result interface{}) {
	c.result = result
}

func (c *Command) ReceiveResult(
	result interface{},
	strict bool,
	greedy bool,
	ctx    HandleContext,
) (accepted bool) {
	if result != nil {
		c.results = append(c.results, result)
		c.result = nil
		return true
	}
	return false
}

func (c *Command) Dispatch(
	handler interface{},
	greedy  bool,
	ctx     HandleContext,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(c.Policy(), handler, c.callback, c, greedy, ctx, c).
		OtherwiseHandled(len(c.results) > count)
}