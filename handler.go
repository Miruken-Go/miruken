package miruken

import (
	"fmt"
)

type (
	// Handler is the uniform metaphor for processing.
	Handler interface {
		Handle(
			callback any,
			greedy   bool,
			composer Handler,
		) HandleResult
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
		Reason  error
	}

	// handlerAdapter adapts an ordinary type to a Handler.
	handlerAdapter struct {
		handler any
	}
)


// handlerAdapter

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
	if IsNil(e.Reason) {
		return e.Message
	}
	return fmt.Sprintf("%v: %s", e.Message, e.Reason.Error())
}

func (e *CanceledError) Unwrap() error {
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
	return builder.WithCallback(callback).New().
		Dispatch(handler, greedy, composer)
}
