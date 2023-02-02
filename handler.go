package miruken

import (
	"fmt"
)

// Handler is the uniform metaphor for processing.
type Handler interface {
	Handle(
		callback any,
		greedy   bool,
		composer Handler,
	) HandleResult
}

// handlerAdapter adapts an ordinary type to a Handler.
type handlerAdapter struct {
	handler any
}

func (h handlerAdapter) Handle(
	callback any,
	greedy   bool,
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

// NotHandledError reports a failed callback.
type NotHandledError struct {
	Callback any
}

func (e *NotHandledError) Error() string {
	return fmt.Sprintf("unhandled %v", e.Callback)
}

// RejectedError reports a rejected callback.
type RejectedError struct {
	Callback any
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("callback %+v was rejected", e.Callback)
}

// CancelledError reports a cancelled operation.
type CancelledError struct {
	Message string
	Reason  error
}

func (e *CancelledError) Error() string {
	if IsNil(e.Reason) {
		return e.Message
	}
	return fmt.Sprintf("%v: %s", e.Message, e.Reason.Error())
}

func (e *CancelledError) Unwrap() error {
	return e.Reason
}

func DispatchCallback(
	handler  any,
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if IsNil(handler) {
		return NotHandled
	}
	switch d := callback.(type) {
	case customizeDispatch:
		return d.Dispatch(handler, greedy, composer)
	case suppressDispatch:
		return NotHandled
	}
	var builder HandlesBuilder
	return builder.
		WithCallback(callback).
		NewHandles().
		Dispatch(handler, greedy, composer)
}
