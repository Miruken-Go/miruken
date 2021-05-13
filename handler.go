package miruken

// handler

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

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return &handlerAdapter{handler}
	}
}