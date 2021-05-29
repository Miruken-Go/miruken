package miruken

import (
	"reflect"
)

type (
	Callback interface {
		ResultType() reflect.Type
		SetResult(result interface{})
		Result() interface{}
	}

	CallbackBase struct {
		many    bool
		results []interface{}
		result  interface{}
	}
)

func (c *CallbackBase) Many() bool {
	return c.many
}

func (c *CallbackBase) ResultType() reflect.Type {
	return nil
}

func (c *CallbackBase) Result() interface{} {
	if result := c.result; result == nil {
		if c.many {
			if c.results == nil {
				c.results = make([]interface{}, 0, 0)
			}
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

func (c *CallbackBase) SetResult(result interface{}) {
	c.result = result
}

func (c *CallbackBase) CopyResult(target interface{}) {
	if c.Many() {
		CopySliceIndirect(c.Result().([]interface{}), target)
	} else {
		CopyIndirect(c.Result(), target)
	}
}

type CallbackDispatcher interface {
	Policy() Policy

 	Dispatch(
		handler  interface{},
		greedy   bool,
		composer Handler,
	) HandleResult
}

type CallbackGuard interface {
	CanDispatch(
		handler interface{},
		binding Binding,
	) (reset func (rawCallback interface{}), approved bool)
}

type ResultReceiver interface {
	ReceiveResult(
		results  interface{},
		strict   bool,
		greedy   bool,
		composer Handler,
	) (accepted bool)
}

type ResultReceiverFunc func(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool)

func (f ResultReceiverFunc) ReceiveResult(
	results  interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return f(results, strict, greedy, composer)
}