package miruken

import (
	"fmt"
	"reflect"
)

func Resolve(handler Handler, target interface{}) error {
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
	inquiry := NewInquiry(typ.Elem(), false, nil)
	if result := handler.Handle(inquiry, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return nil
	}
	if result := inquiry.Result(); result != nil {
		resultValue := reflect.ValueOf(result)
		if resultType := resultValue.Type(); resultType.AssignableTo(typ.Elem()) {
			val.Elem().Set(resultValue)
		} else {
			panic(fmt.Sprintf("*target must be assignable to %v", typ))
		}
	}
	return nil
}

func ResolveAll(handler Handler, target interface{}) error {
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
	inquiry := NewInquiry(typ.Elem(), false, nil)
	if result := handler.Handle(inquiry, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return nil
	}
	if results := inquiry.Result(); results != nil {
		if source, ok := results.([]interface{}); ok {
			CopyTypedSlice(source, target)
		} else {
			panic(fmt.Sprintf("expected slice result, found %#v", results))
		}
	}
	return nil
}
