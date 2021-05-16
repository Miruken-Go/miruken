package miruken

func Handle(handler Handler, callback interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if result := handler.Handle(callback, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return &NotHandledError{callback}
	}
	return nil
}

func HandleAll(handler Handler, callback interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if result := handler.Handle(callback, true, nil); result.IsError() {
		return result.Error()
	}  else if !result.handled {
		return &NotHandledError{callback}
	}
	return nil
}

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return &handlerAdapter{handler}
	}
}
