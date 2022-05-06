package miruken

import "reflect"

func MakeCaller(fun any) (func(Handler) []reflect.Value, error) {
	if fun == nil {
		panic("fun cannot be nil")
	}
	val := reflect.ValueOf(fun)
	if typ := val.Type(); typ.Kind() != reflect.Func {
		panic("fun is not a valid function")
	} else {
		numArgs := typ.NumIn()
		args    := make([]arg, numArgs)
		if err := buildDependencies(typ, 0, numArgs, args, 0); err != nil {
			return nil, err
		}
		return func(handler Handler) []reflect.Value {
			return nil
		}, nil
	}
}
