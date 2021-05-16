package miruken

import "fmt"

// Handler

type Handler interface {
	Handle(
		callback interface{},
		greedy   bool,
		ctx      HandleContext,
	) HandleResult
}

// handlerAdapter

type handlerAdapter struct {
	handler interface{}
}

func (h *handlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {
	return DispatchCallback(h.handler, callback, greedy, ctx)
}

// NotHandledError

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
	ctx      HandleContext,
) HandleResult {
	if handler == nil {
		return NotHandled
	}
	if dispatch, ok := callback.(CallbackDispatcher); ok {
		return dispatch.Dispatch(handler, greedy, ctx)
	}
	command := &Command{callback: callback}
	return command.Dispatch(handler, greedy, ctx)
}
