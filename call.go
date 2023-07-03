package miruken

import (
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type CallerFunc func(Handler, ...any) ([]any, *promise.Promise[[]any], error)

func MakeCaller(fun any) (CallerFunc, error) {
	if fun == nil {
		panic("fun cannot be nil")
	}
	val, ok := fun.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(fun)
	}
	if typ := val.Type(); typ.Kind() != reflect.Func {
		panic("fun is not a valid function")
	} else {
		numArgs := typ.NumIn()
		args    := make([]arg, numArgs)
		if err := buildDependencies(typ, 0, numArgs, args, 0); err != nil {
			return nil, err
		}
		return func(handler Handler, initArgs ...any) ([]any, *promise.Promise[[]any], error) {
			ctx := HandleContext{composer: handler}
			return callFunc(val, ctx, args[len(initArgs):], initArgs...)
		}, nil
	}
}
