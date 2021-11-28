package miruken

import (
	"fmt"
	"reflect"
)

func IsNil(val interface{}) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Type().Kind() {
	case reflect.Chan,
		 reflect.Func,
		 reflect.Interface,
		 reflect.Map,
		 reflect.Ptr,
		 reflect.Slice:
			 return v.IsNil()
	default:
		return false
	}
}

// TargetValue validates the interface contains a
// non-nil typed pointer and return reflect.Value.
func TargetValue(target interface{}) reflect.Value {
	if target == nil {
		panic("target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("target must be a non-nil pointer")
	}
	return val
}

// TargetSliceValue validates the interface contains a
// non-nil typed slice pointer and return reflect.Value.
func TargetSliceValue(target interface{}) reflect.Value {
	val := TargetValue(target)
	typ := val.Type()
	if typ.Elem().Kind() != reflect.Slice {
		panic("target must be a non-nil slice pointer")
	}
	return val
}

// CopyIndirect copies the contents of src into the
// target pointer or reflect.Value.
func CopyIndirect(src interface{}, target interface{}) {
	var val reflect.Value
	if v, ok := target.(reflect.Value); ok {
		val = v
	} else {
		val = TargetValue(target)
	}
	val = reflect.Indirect(val)
	typ := val.Type()
	if src != nil {
		resultValue := reflect.ValueOf(src)
		if resultType := resultValue.Type(); resultType.AssignableTo(typ) {
			val.Set(resultValue)
		} else {
			panic(fmt.Sprintf("%T must be assignable to %v", src, typ))
		}
	} else {
		val.Set(reflect.Zero(typ))
	}
}

// CopySliceIndirect copies the contents of src slice into
// the target pointer or reflect.Value.
func CopySliceIndirect(src []interface{}, target interface{}) {
	var val reflect.Value
	if v, ok := target.(reflect.Value); ok {
		val = v
	} else {
		val = TargetSliceValue(target)
	}
	val = reflect.Indirect(val)
	typ := val.Type()
	elementType := typ.Elem()
	if src == nil {
		val.Set(reflect.MakeSlice(typ, 0, 0))
		return
	}
	slice := reflect.MakeSlice(typ, len(src), len(src))
	for i, element := range src {
		if reflect.TypeOf(element).AssignableTo(elementType) {
			slice.Index(i).Set(reflect.ValueOf(element))
		} else {
			panic(fmt.Sprintf(
				"%T at index %v must be assignable to %v",
				element, i, elementType))
		}
	}
	val.Set(slice)
}

func forEach(iter interface{}, f func(i int, val interface{})) {
	v := reflect.ValueOf(iter)
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
		panic(fmt.Errorf("forEach: expected iter or array, found %q", v.Kind().String()))
	}
}

func coerceToPtr(
	givenType   reflect.Type,
	desiredType reflect.Type,
) reflect.Type {
	if givenType.AssignableTo(desiredType) {
		return givenType
	} else if givenType.Kind() != reflect.Ptr {
		givenType = reflect.PtrTo(givenType)
		if givenType.AssignableTo(desiredType) {
			return givenType
		}
	}
	return nil
}

func newWithTag(
	typ reflect.Type,
	tag reflect.StructTag,
) (interface{}, error) {
	var val interface{}
	if typ.Kind() == reflect.Ptr {
		val = reflect.New(typ.Elem()).Interface()
	} else {
		val = reflect.New(typ).Elem().Interface()
	}
	if len(tag) > 0 {
		if init, ok := val.(interface {
			InitWithTag(reflect.StructTag) error
		}); ok {
			if err := init.InitWithTag(tag); err != nil {
				return val, err
			}
			return val, nil
		}
	}
	if init, ok := val.(interface {
		Init() error
	}); ok {
		if err := init.Init(); err != nil {
			return val, err
		}
	}
	return val, nil
}