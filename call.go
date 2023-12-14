package miruken

import (
	"reflect"

	"github.com/miruken-go/miruken/promise"
)

// CallerFunc is a function that invokes a handler method.
// The first argument is the handler instance.
// The remaining arguments become the initial arguments to the handler method.
// All remaining handler method arguments are resolved from the context.
type CallerFunc func(Handler, ...any) ([]any, *promise.Promise[[]any], error)

// MakeCaller creates a CallerFunc from a function.
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
		args := make([]arg, numArgs)
		if err := buildDependencies(typ, 0, numArgs, args, 0); err != nil {
			return nil, err
		}
		return func(handler Handler, initArgs ...any) ([]any, *promise.Promise[[]any], error) {
			fun := funcCall{val, args[len(initArgs):]}
			return fun.Invoke(HandleContext{Composer: handler}, initArgs...)
		}, nil
	}
}
