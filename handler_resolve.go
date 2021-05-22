package miruken

func Resolve(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv       := TargetValue(target)
	inquiry  := NewInquiry(tv.Type().Elem(), false, nil)
	if result := handler.Handle(inquiry, false, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
	return nil
}

func ResolveAll(handler Handler, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	tv      := TargetSliceValue(target)
	inquiry := NewInquiry(tv.Type().Elem().Elem(), true, nil)
	if result := handler.Handle(inquiry, true, nil); result.IsError() {
		return result.Error()
	}
	inquiry.CopyResult(tv)
	return nil
}
