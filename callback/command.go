package callback

import (
	"reflect"
)

type Command struct {
	callback interface{}
	many     bool
	results  []interface{}
	result   interface{}
}

func (c *Command) IsMany() bool {
	return c.many
}

func (c *Command) GetCallback() interface{} {
	return c.callback
}

func (c *Command) GetPolicy() Policy {
	return GetHandlesPolicy()
}

func (c *Command) GetResultType() reflect.Type {
	return nil
}

func (c *Command) GetResult() interface{} {
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
	result  interface{},
	strict  bool,
	greedy  bool,
	context HandleContext,
) bool {
	if result != nil {
		c.results = append(c.results, result)
		c.result = nil
		return true
	}
	return false
}

func (c *Command) Dispatch(
	handler  interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	count := len(c.results)
	return DispatchPolicy(c.GetPolicy(), handler, c, greedy, context, c).
		OtherwiseHandled(len(c.results) > count)
}