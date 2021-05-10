package callback

// Handler

type Handler interface {
	Handle(
		callback interface{},
		greedy   bool,
		context  HandleContext,
	) (HandleResult, error)
}

// HandlerAdapter

type HandlerAdapter struct {
	Handler interface{}
}

func (h *HandlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) (HandleResult, error) {
	return DispatchCallback(h.Handler, callback, greedy, context)
}

func DispatchCallback(
	handler  interface{},
	callback interface{},
	greedy   bool,
	context  HandleContext,
) (HandleResult, error) {
	if handler == nil {
		return NotHandled, nil
	}
	if dispatch, ok := callback.(CallbackDispatcher); ok {
		return dispatch.Dispatch(handler, greedy, context)
	}
	command := &Command{Callback: callback}
	return command.Dispatch(handler, greedy, context)
}
