package miruken

import (
	"fmt"
	"reflect"
)

// NotHandledError

type NotHandledError struct {
	callback interface{}
}

func (e *NotHandledError) Error() string {
	return fmt.Sprintf("callback %#v not handled", e.callback)
}

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

func Invoke(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("target must be a non-nil pointer")
	}
	command := &Command{callback: callback}
	if result := handler.Handle(command, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return &NotHandledError{callback}
	}
	if result := command.Result(); result != nil {
		resultValue := reflect.ValueOf(result)
		if resultType := resultValue.Type(); resultType.AssignableTo(typ.Elem()) {
			val.Elem().Set(resultValue)
		} else {
			panic(fmt.Sprintf("*target must be assignable to %v", typ))
		}
	}
	return nil
}
