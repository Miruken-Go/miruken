package miruken

import "fmt"

// Handler is the uniform metaphor for processing
type Handler interface {
	Handle(
		callback interface{},
		greedy   bool,
		composer Handler,
	) HandleResult
}

// handlerAdapter adapts an ordinary type to a Handler
type handlerAdapter struct {
	handler interface{}
}

func (h handlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h.handler, callback, greedy, composer)
}

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return handlerAdapter{handler}
	}
}

// NotHandledError reports a failed callback
type NotHandledError struct {
	callback interface{}
}

func (e NotHandledError) Error() string {
	return fmt.Sprintf("callback %#v not handled", e.callback)
}

// RejectedError reports a rejected callback
type RejectedError struct {
	callback interface{}
}

func (e RejectedError) Error() string {
	return fmt.Sprintf("callback %#v was rejected", e.callback)
}

func DispatchCallback(
	handler  interface{},
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if handler == nil {
		return NotHandled
	}
	switch d := callback.(type) {
	case CallbackDispatcher:
		return d.Dispatch(handler, greedy, composer)
	case SuppressDispatch:
		return NotHandled
	}
	return new(CommandBuilder).
		WithCallback(callback).
		NewCommand().
		Dispatch(handler, greedy, composer)
}
