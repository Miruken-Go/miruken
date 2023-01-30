package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/slices"
	"reflect"
)

type (
	// Void is friendly name for nothing.
	Void = struct {}

	// FuncBinder creates a binding to a function.
	FuncBinder interface {
		NewFuncBinding(
			fun  reflect.Value,
			spec *bindingSpec,
			key  any,
		) (Binding, error)
	}

	// FuncBinding models a `key` Binding to a function.
	FuncBinding struct {
		BindingBase
		key  any
		fun  reflect.Value
		args []arg
	}

	// FuncBindingError reports a failed function binding.
	FuncBindingError struct {
		fun    reflect.Value
		reason error
	}
)

func (b *FuncBinding) Key() any {
	return b.key
}

func (b *FuncBinding) Invoke(
	ctx      HandleContext,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	return callFunc(b.fun, ctx, b.args, initArgs...)
}

func (e *FuncBindingError) Func() reflect.Value {
	return e.fun
}

func (e *FuncBindingError) Error() string {
	return fmt.Sprintf("invalid function %v: %v", e.fun, e.reason)
}

func (e *FuncBindingError) Unwrap() error {
	return e.reason
}

// callFunc calls the function stored in the fun argument.
// It resolves the args input and appends them to the initArgs
// to supply the input arguments to the function.
// It returns the output results slice and/or and error if not
// asynchronous or a promise to the output results.
func callFunc(
	fun      reflect.Value,
	ctx      HandleContext,
	args     []arg,
	initArgs ...any,
) ([]any, *promise.Promise[[]any], error) {
	fromIndex := len(initArgs)
	if resolvedArgs, pa, err := resolveArgs(fun.Type(), fromIndex, args, ctx); err != nil {
		return nil, nil, err
	} else if pa == nil {
		return callFuncWithArgs(fun, initArgs, resolvedArgs), nil, nil
	} else {
		return nil, promise.Then(pa, func(ra []reflect.Value) []any {
			return callFuncWithArgs(fun, initArgs, ra)
		}), nil
	}
}

// callFuncWithArgs calls the function stored in the fun argument.
// It combines the initArgs and resolvedArgs to supply the input
// arguments to the function.
// It returns the output results slice.
func callFuncWithArgs(
	fun          reflect.Value,
	initArgs     []any,
	resolvedArgs []reflect.Value,
) []any {
	initCount := len(initArgs)
	in := make([]reflect.Value, len(initArgs) + len(resolvedArgs))
	for i, ia := range initArgs {
		in[i] = reflect.ValueOf(ia)
	}
	for i, aa := range resolvedArgs {
		in[initCount + i] = aa
	}
	return slices.Map[reflect.Value, any](fun.Call(in),
		func(v reflect.Value) any { return v.Interface() })
}

// mergeOutput analyzes the standard function return values and
// normalizes them to produce immediate or asynchronous results.
// If an error is present it is returned immediately.
// If not asynchronous (2nd output is nil) and
//   - last output is an error, return it immediately
//   - first output is a promise, resolve and replace the
//     first output element
// Otherwise, if asynchronous (2nd  output is promise), resolve
// and repeat steps above.
// It returns the normalized output.
func mergeOutput(
	out  []any,
	pout *promise.Promise[[]any],
	err  error,
) ([]any, *promise.Promise[[]any], error) {
	if err != nil {
		// if error, fail early
		return out, pout, err
	}
	if pout == nil {
		if len(out) > 0 {
			// if function error (last output), fail early
			if e, ok := out[len(out)-1].(error); ok {
				return nil, nil, e
			} else if pf, ok := out[0].(promise.Reflect); ok {
				// if first output is a promise. resolve and replace
				return nil, promise.Coerce[[]any](
					pf.Then(func(first any) any {
						oo := make([]any, len(out))
						copy(oo, out)
						oo[0] = first
						return oo
					})), nil
			}
		}
		return out, nil, nil
	}
	// if promise, resolve and check output
	return out, promise.Then(pout, func(oo []any) []any {
		if len(oo) > 0 {
			// if function error, panic
			if e, ok := oo[len(oo)-1].(error); ok {
				panic(e)
			} else if pf, ok := oo[0].(promise.Reflect); ok {
				// if first output is a promise. await and replace
				if first, err := pf.AwaitAny(); err != nil {
					panic(err)
				} else {
					oo[0] = first
				}
			}
		}
		return oo
	}), nil
}

// mergeOutputAwait is similar to mergeOutput but is expected to be
// called in the context of an asynchronous operation (goroutine)
// from any call to a promise (New, Then, Catch, ...).
// Since this call is already in a goroutine, it will not block the
// initial operation and can use Await to obtain the intermediate
// results.  It can be used in Filter's that perform asynchronous
// operations and want to normalize outputs.
func mergeOutputAwait(
	out  []any,
	pout *promise.Promise[[]any],
	err  error,
) []any {
	if err != nil {
		// if error, fail early
		panic(err)
	}
	if pout == nil {
		if len(out) > 0 {
			// if function error (last output), panic
			if err, ok := out[len(out)-1].(error); ok {
				panic(err)
			} else if pf, ok := out[0].(promise.Reflect); ok {
				// if first output is a promise. await and replace
				if first, err := pf.AwaitAny(); err != nil {
					panic(err)
				} else {
					oo := make([]any, len(out))
					copy(oo, out)
					oo[0] = first
					return oo
				}
			}
		}
		return out
	}
	// if promise, await and check output
	if oo, err := pout.Await(); err != nil {
		panic(err)
	} else if len(oo) > 0 {
		// if function error (last output), panic
		if err, ok := oo[len(oo)-1].(error); ok {
			panic(err)
		} else if pf, ok := oo[0].(promise.Reflect); ok {
			// if first output is a promise. await and replace
			if first, err := pf.AwaitAny(); err != nil {
				panic(err)
			} else {
				oo[0] = first
			}
		}
		return oo
	} else {
		return oo
	}
}