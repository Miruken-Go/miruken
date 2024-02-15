package internal

import "reflect"

var (
	AnyType      = reflect.TypeFor[any]()
	AnySliceType = reflect.TypeFor[[]any]()
	ErrorType    = reflect.TypeFor[error]()
)
