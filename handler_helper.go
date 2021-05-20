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

func With(handler Handler, values ... interface{}) Handler {
	if handler == nil {
		return nil
	}
	var valueHandlers []interface{}
	for _, val := range values {
		if val != nil {
			valueHandlers = append(valueHandlers, NewProvider(val))
		}
	}
	if len(valueHandlers) > 0 {
		return AddHandlers(handler, valueHandlers...)
	}
	return handler
}

func ToHandler(handler interface{}) Handler {
	switch h := handler.(type) {
	case Handler: return h
	default: return &handlerAdapter{handler}
	}
}
