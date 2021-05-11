package callback

import (
	"reflect"
)

type Command struct {
	Callback interface{}
	Many     bool
	Results  []interface{}
	Result   interface{}
}

func (c *Command) GetPolicy() Policy {
	return HandlesPolicy
}

func (c *Command) GetResultType() reflect.Type {
	return nil
}

func (c *Command) GetResult() interface{} {
	if result := c.Result; result == nil {
		if c.Many {
			c.Result = c.Results
		} else {
			if len(c.Results) == 0 {
				c.Result = nil
			} else {
				c.Result = c.Results[0]
			}
		}
	}
	return c.Result
}

func (c *Command) SetResult(result interface{}) {
	c.Result = result
}

func (c *Command) Respond(
	result   interface{},
	strict   bool,
	context  HandleContext,
) {

}

func (c *Command) Dispatch(
	handler  interface{},
	greedy   bool,
	context  HandleContext,
)HandleResult {
	return NotHandled
}