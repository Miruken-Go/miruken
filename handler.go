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

func (h *handlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchCallback(h.handler, callback, greedy, composer)
}

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return &handlerAdapter{handler}
	}
}

// NotHandledError reports a callback failure
type NotHandledError struct {
	callback interface{}
}

func (e *NotHandledError) Error() string {
	return fmt.Sprintf("callback %#v not handled", e.callback)
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
