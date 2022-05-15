package miruken

import (
	"reflect"
)

type (
	// Callback represents any action.
	Callback interface {
		Key() any
		Source() any
		Policy() Policy
		ResultType() reflect.Type
		Result(many bool) any
		SetResult(result any)
		ReceiveResult(
			result   any,
			strict   bool,
			composer Handler,
		) HandleResult
	}

	// AcceptResultFunc accepts or rejects callback results.
	AcceptResultFunc func (
		result   any,
		composer Handler,
	) HandleResult

	// CallbackBase is abstract Callback implementation.
	CallbackBase struct {
		results []any
		result  any
		accept  AcceptResultFunc
	}
)

func (c *CallbackBase) Source() any {
	return nil
}

func (c *CallbackBase) ResultType() reflect.Type {
	return nil
}

func (c *CallbackBase) Result(many bool) any {
	if result := c.result; result == nil {
		if many {
			if c.results == nil {
				c.results = make([]any, 0, 0)
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

func (c *CallbackBase) SetResult(result any) {
	c.result = result
}

func (c *CallbackBase) AddResult(
	result   any,
	composer Handler,
) HandleResult {
	if IsNil(result) {
		return NotHandled
	}
	if c.accept != nil {
		return c.accept(result, composer)
	}
	c.results = append(c.results, result)
	c.result  = nil
	return Handled
}

func (c *CallbackBase) ReceiveResult(
	result   any,
	strict   bool,
	composer Handler,
) HandleResult {
	return c.AddResult(result, composer)
}

func (c *CallbackBase) CopyResult(target any, many bool) {
	if many {
		CopySliceIndirect(c.Result(true).([]any), target)
	} else {
		CopyIndirect(c.Result(false), target)
	}
}

type CallbackBuilder struct {
	accept AcceptResultFunc
}

func (b *CallbackBuilder) WithAcceptResult(
	accept AcceptResultFunc,
) *CallbackBuilder {
	b.accept = accept
	return b
}

func (b *CallbackBuilder) CallbackBase() CallbackBase {
	return CallbackBase{accept: b.accept}
}

// customizeDispatch marks customized Callback dispatch.
type customizeDispatch interface {
 	Dispatch(
		handler  any,
		greedy   bool,
		composer Handler,
	) HandleResult
}

// suppressDispatch marks a type that opts out of Callback dispatch.
type suppressDispatch interface {
	SuppressDispatch()
}

// CallbackGuard detects and prevents circular Callback dispatch.
type CallbackGuard interface {
	CanDispatch(
		handler any,
		binding Binding,
	) (reset func (), approved bool)
}
