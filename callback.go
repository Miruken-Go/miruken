package miruken

import (
	"reflect"
)

type (
	// Callback models an action.
	Callback interface {
		Policy() Policy
		Key() interface{}
		ResultType() reflect.Type
		Result() interface{}
		SetResult(result interface{})
	}

	// CallbackBase is abstract Callback implementation.
	CallbackBase struct {
		many    bool
		results []interface{}
		result  interface{}
		accept  AcceptResult
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

func (c *CallbackBase) AcceptResult(
	result   interface{},
	greedy   bool,
	composer Handler,
) bool {
	if c.accept != nil {
		return c.accept.Accept(result, greedy, composer)
	}
	return true
}

type CallbackBuilder struct {
	many   bool
	accept AcceptResult
}

func (b *CallbackBuilder) WithMany() *CallbackBuilder {
	b.many = true
	return b
}

func (b *CallbackBuilder) WithAcceptResult(
	accept AcceptResult,
) *CallbackBuilder {
	b.accept = accept
	return b
}

func (b *CallbackBuilder) Callback() CallbackBase {
	return CallbackBase{many: b.many, accept: b.accept}
}

// CallbackDispatcher allows customized Callback dispatch.
type CallbackDispatcher interface {
 	Dispatch(
		handler  interface{},
		greedy   bool,
		composer Handler,
	) HandleResult
}

// SuppressDispatch marks a type that should not perform dispatching.
type SuppressDispatch interface {
	suppressDispatch()
}

// CallbackGuard prevents circular actions.
type CallbackGuard interface {
	CanDispatch(
		handler interface{},
		binding Binding,
	) (reset func (), approved bool)
}

// ResultReceiver defines acceptance criteria of results.
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

type AcceptResult interface {
	Accept(
		result   interface{},
		greedy   bool,
		composer Handler,
	) bool
}

type AcceptResultFunc func (
	result   interface{},
	greedy   bool,
	composer Handler,
) bool

func (f AcceptResultFunc) Accept(
	result   interface{},
	greedy   bool,
	composer Handler,
) bool { return f(result, greedy, composer)}
