package miruken

import (
	"fmt"
	"reflect"
)

func forEach(iter any, f func(i int, val any) bool) {
	v := reflect.ValueOf(iter)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i).Interface()
			if f(i, val) {
				return
			}
		}
	default:
		panic(fmt.Errorf("forEach: expected iter or array, found %q", v.Kind().String()))
	}
}
