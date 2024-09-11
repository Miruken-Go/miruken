package miruken

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

// InvariantPolicy matches equal input values.
type InvariantPolicy struct {
	FilterScope
}

var ErrInvMissingCallback = errors.New("invariant: missing callback argument")

func (p *InvariantPolicy) VariantKey(
	key any,
) (variant, unknown bool) {
	return false, false
}

func (p *InvariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches, exact bool) {
	if key == otherKey {
		return true, true
	}
	return false, false
}

func (p *InvariantPolicy) Strict() bool {
	return true
}

func (p *InvariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	return false
}

func (p *InvariantPolicy) AcceptResults(
	results []any,
) (any, HandleResult, []Effect, []any) {
	switch len(results) {
	case 0:
		return nil, Handled, nil, nil
	case 1:
		switch result := results[0].(type) {
		case error:
			return nil, NotHandled.WithError(result), nil, nil
		case HandleResult:
			return nil, result, nil, nil
		default:
			if effect, _ := MakeEffect(result, false); effect != nil {
				return nil, Handled, []Effect{effect}, nil
			}
			return result, Handled, nil, nil
		}
	default:
		iEnd := 0
		hr   := Handled
		switch err := results[len(results)-1].(type) {
		case error:
			return nil, NotHandled.WithError(err), nil, nil
		case HandleResult:
			if !err.Handled() || err.IsError() {
				return nil, err, nil, nil
			}
			hr = err
			iEnd++
		}
		var res any
		iStart := 0
		first  := results[0]
		if h, ok := first.(HandleResult); ok {
			hr = hr.And(h)
			if !hr.Handled() || hr.IsError() {
				return nil, hr, nil, nil
			}
			iStart++
		} else {
			if effect, _ := MakeEffect(first, false); effect == nil {
				res = first
				iStart++
			}
		}
		effects, xs, err := MakeEffects(false, results[iStart:len(results)-iEnd])
		if err != nil {
			return res, NotHandled.WithError(err), nil, nil
		}
		return res, hr, effects, xs
	}
}

func (p *InvariantPolicy) NewMethodBinding(
	method *reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, k, err := validateInvariantFunc(method.Type, spec, key, 1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &MethodBinding{
			funcCall{method.Func, args},
			BindingBase{
				FilterScope{spec.filters},
				spec.flags, spec.metadata,
			}, k, *method, spec.lt,
		}, nil
	}
}

func (p *InvariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, k, err := validateInvariantFunc(fun.Type(), spec, key, 0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &FuncBinding{
			funcCall{fun, args},
			BindingBase{
				FilterScope{spec.filters},
				spec.flags, spec.metadata,
			}, k, spec.lt,
		}, nil
	}
}

func validateInvariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, ck any, err error) {
	ck = key
	numArgs := funType.NumIn()
	numOut := funType.NumOut()
	args = make([]arg, numArgs-skip)
	args[0] = spec.arg
	index := 1

	// Source argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1 + skip); arg.AssignableTo(callbackType) {
			args[1] = CallbackArg{}
			if ck == nil {
				ck = internal.AnyType
			}
		} else {
			if ck == nil {
				ck = arg
			}
			args[1] = sourceArg{}
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		err = ErrInvMissingCallback
	} else if ck == nil {
		ck = internal.AnyType
	}

	if err2 := buildDependencies(funType, index+skip, numArgs, args, index); err2 != nil {
		err = errors.Join(err, fmt.Errorf("invariant: %w", err2))
	}

	resIdx := -1

	for i := range numOut {
		out := funType.Out(i)
		if out.AssignableTo(internal.ErrorType) {
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"invariant: error found at index %v must be last return", i))
			}
		} else if out.AssignableTo(handleResType) {
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"invariant: HandleResult found at index %v must be last return", i))
			}
		} else if ok, err2 := ValidEffect(out); ok {
			// ignore effects
		} else if err2 != nil {
			err = errors.Join(err, fmt.Errorf(
				"invariant: invalid effect at index %v: %w", i, err2))
		} else if resIdx == -1 {  // response assumed be first
			resIdx = i
			if lt, ok := promise.Inspect(out); ok {
				spec.flags |= bindingAsync
				out = lt
			}
			spec.setLogicalOutputType(out)
		}
	}
	return
}
