package miruken

import (
	"fmt"
	"reflect"
)

func CopyTypedSlice(source []interface{}, targetPtr interface{}) {
	val := reflect.ValueOf(targetPtr)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Slice || val.IsNil() {
		panic("target must be a non-nil slice pointer")
	}
	sliceType   := typ.Elem()
	elementType := sliceType.Elem()
	slice       := reflect.MakeSlice(sliceType, len(source), len(source))
	for i, element := range source {
		if reflect.TypeOf(element).AssignableTo(elementType) {
			slice.Index(i).Set(reflect.ValueOf(element))
		} else {
			panic(fmt.Sprintf("element at index %v must be assignable to %v", i, elementType))
		}
	}
	val.Elem().Set(slice)
}

func forEach(iterable interface{}, f func(i int, val interface{})) {
	v := reflect.ValueOf(iterable)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i).Interface()
			f(i, val)
		}
	default:
		panic(fmt.Errorf("forEach: expected iterable or array, found %q", v.Kind().String()))
	}
}