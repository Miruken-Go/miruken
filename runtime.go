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
