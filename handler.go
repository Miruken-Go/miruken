package miruken

import (
	"fmt"

	"github.com/miruken-go/miruken/internal"
)

type (
	// Handler is the uniform metaphor for processing.
	Handler interface {
		Handle(
			callback any,
			greedy bool,
			composer Handler,
		) HandleResult
	}

	// HandleContext allows interrogation of the current Callback.
	HandleContext struct {
		Handler  any
		Callback Callback
		Binding  Binding
		Composer Handler
		Greedy   bool
	}

	// NotHandledError reports a failed callback.
	NotHandledError struct {
		Callback any
	}

	// RejectedError reports a rejected callback.
	RejectedError struct {
		Callback any
	}

	// CanceledError reports a canceled operation.
	CanceledError struct {
		Message string
		Cause   error
	}

	// handlerAdapter adapts an ordinary type to a Handler.
	handlerAdapter struct {
		handler any
	}
)

// HandleContext

func (c HandleContext) Handle(
	callback any,
	greedy bool,
	composer Handler,
) HandleResult {
	return c.Composer.Handle(callback, greedy, composer)
}

// handlerAdapter

func (h handlerAdapter) Handle(
	callback any,
	greedy bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h.handler, callback, greedy, composer)
}

func ToHandler(handler any) Handler {
	switch h := handler.(type) {
	case Handler:
		return h
	default:
		return handlerAdapter{handler}
	}
}

// NotHandledError

func (e *NotHandledError) Error() string {
	return fmt.Sprintf("unhandled \"%T\"", e.Callback)
}

// RejectedError

func (e *RejectedError) Error() string {
	return fmt.Sprintf("callback \"%T\" was rejected", e.Callback)
}

// CanceledError

func (e *CanceledError) Error() string {
	if internal.IsNil(e.Cause) {
		return e.Message
	}
	return fmt.Sprintf("%v: %s", e.Message, e.Cause.Error())
}

func (e *CanceledError) Unwrap() error {
	return e.Cause
}

func DispatchCallback(
	handler any,
	callback any,
	greedy bool,
	composer Handler,
) HandleResult {
	if internal.IsNil(handler) {
		return NotHandled
	}
	switch d := callback.(type) {
	case customizeDispatch:
		return d.Dispatch(handler, greedy, composer)
	case suppressDispatch:
		return NotHandled
	}
	var builder HandlesBuilder
	return builder.WithCallback(callback).New().
		Dispatch(handler, greedy, composer)
}
