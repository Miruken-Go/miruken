package runtime

import "reflect"

func GetField(value interface{}, field string) reflect.Value {
	r := reflect.ValueOf(value)
	return reflect.Indirect(r).FieldByName(field)
}
