package miruken

import (
	"fmt"
	"reflect"
)

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

func InvokeAll(handler Handler, callback interface{}, target interface{}) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Slice || val.IsNil() {
		panic("target must be a non-nil slice pointer")
	}
	command := &Command{callback: callback, many: true}
	if result := handler.Handle(command, true, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return &NotHandledError{callback}
	}
	if results := command.Result(); results != nil {
		if source, ok := results.([]interface{}); ok {
			CopyTypedSlice(source, target)
		} else {
			panic(fmt.Sprintf("expected slice result, found %#v", results))
		}
	}
	return nil
}
