package miruken

import "fmt"

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
	callback any
}

func (e NotHandledError) Callback() any {
	return e.callback
}

func (e NotHandledError) Error() string {
	return fmt.Sprintf("callback %#v not handled", e.callback)
}

// RejectedError reports a rejected callback.
type RejectedError struct {
	callback any
}

func (e RejectedError) Callback() any {
	return e.callback
}

func (e RejectedError) Error() string {
	return fmt.Sprintf("callback %#v was rejected", e.callback)
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
