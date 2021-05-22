package miruken

func Invoke(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetValue(target)
	command := NewCommand(callback, false)
	if result := handler.Handle(command, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return &NotHandledError{callback}
	}
	command.CopyResult(tv)
	return nil
}

func InvokeAll(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	command := NewCommand(callback, true)
	if result := handler.Handle(command, true, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return &NotHandledError{callback}
	}
	command.CopyResult(tv)
	return nil
}
