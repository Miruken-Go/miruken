package callback

// Handler

type Handler interface {
	Handle(
		callback interface{},
		greedy   bool,
		context  HandleContext,
	) HandleResult
}

// HandlerAdapter

type HandlerAdapter struct {
	Handler interface{}
}

func (h *HandlerAdapter) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	return DispatchCallback(h.Handler, callback, greedy, context)
}

func DispatchCallback(
	handler  interface{},
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {
	if handler == nil {
		return NotHandled
	}
	if dispatch, ok := callback.(CallbackDispatcher); ok {
		return dispatch.Dispatch(handler, greedy, context)
	}
	command := &Command{callback: callback}
	return command.Dispatch(handler, greedy, context)
}

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return &HandlerAdapter{handler}
	}
}