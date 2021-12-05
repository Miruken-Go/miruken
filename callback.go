package miruken

import (
	"reflect"
)

type (
	// Callback represents an action.
	Callback interface {
		Key() interface{}
		Policy() Policy
		ResultType() reflect.Type
		Result() interface{}
		SetResult(result interface{})
		ReceiveResult(
			result   interface{},
			strict   bool,
			greedy   bool,
			composer Handler,
		) (accepted bool)
	}

	// AcceptResultFunc validates callback results.
	AcceptResultFunc func (
		result   interface{},
		greedy   bool,
		composer Handler,
	) bool

	// CallbackBase is abstract Callback implementation.
	CallbackBase struct {
		many    bool
		results []interface{}
		result  interface{}
		accept  AcceptResultFunc
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

func (c *CallbackBase) AddResult(
	result   interface{},
	greedy   bool,
	composer Handler,
) bool {
	if IsNil(result) ||
		(c.accept != nil && !c.accept(result, greedy, composer)) {
		return false
	}
	c.results = append(c.results, result)
	c.result  = nil
	return true
}

func (c *CallbackBase) CopyResult(target interface{}) {
	if c.Many() {
		CopySliceIndirect(c.Result().([]interface{}), target)
	} else {
		CopyIndirect(c.Result(), target)
	}
}

type CallbackBuilder struct {
	many   bool
	accept AcceptResultFunc
}

func (b *CallbackBuilder) WithMany() *CallbackBuilder {
	b.many = true
	return b
}

func (b *CallbackBuilder) WithAcceptResult(
	accept AcceptResultFunc,
) *CallbackBuilder {
	b.accept = accept
	return b
}

func (b *CallbackBuilder) CallbackBase() CallbackBase {
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
