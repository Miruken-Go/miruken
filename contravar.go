package miruken

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

// ContravariantPolicy matches related input values.
type ContravariantPolicy struct {
	FilterScope
}

var ErrConMissingCallback = errors.New("contravariant: missing callback argument")

func (p *ContravariantPolicy) VariantKey(
	key any,
) (variant, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, internal.IsAny(typ)
	}
	return false, false
}

func (p *ContravariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches, exact bool) {
	if key == otherKey {
		return true, true
	} else if invariant {
		return false, false
	} else if bt, isType := key.(reflect.Type); isType {
		if internal.IsAny(bt) {
			return true, false
		} else if kt, isType := otherKey.(reflect.Type); isType {
			if kt.AssignableTo(bt) {
				return true, false
			}
			if kt.Kind() == reflect.Ptr && kt.Elem().AssignableTo(bt) {
				return true, false
			}
		}
	}
	return false, false
}

func (p *ContravariantPolicy) Strict() bool {
	return true
}

func (p *ContravariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), false)
	return !exact && matches
}

func (p *ContravariantPolicy) AcceptResults(
	results []any,
) (any, HandleResult, []Effect, []any) {
	return acceptResultsWithEffects(results)
}

func (p *ContravariantPolicy) NewMethodBinding(
	method *reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, k, err := validateContravariantFunc(method.Type, spec, key, 1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &MethodBinding{
			newFuncCall(method.Func, args),
			spec.bindingBase(), k, *method, spec.lt,
		}, nil
	}
}

func (p *ContravariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, k, err := validateContravariantFunc(fun.Type(), spec, key, 0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &FuncBinding{
			newFuncCall(fun, args),
			spec.bindingBase(), k, spec.lt,
		}, nil
	}
}

//goland:noinspection DuplicatedCode
func validateContravariantFunc(
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
		err = ErrConMissingCallback
	} else if ck == nil {
		ck = internal.AnyType
	}

	if err2 := buildDependencies(funType, index+skip, numArgs, args, index); err2 != nil {
		err = errors.Join(err, fmt.Errorf("contravariant: %w", err2))
	}

	resIdx := -1

	for i := range numOut {
		out := funType.Out(i)
		if out.AssignableTo(internal.ErrorType) {
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"contravariant: error found at index %v must be last return", i))
			}
		} else if out.AssignableTo(handleResType) {
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"contravariant: HandleResult found at index %v must be last return", i))
			}
		} else if ok, err2 := ValidEffect(out); ok {
			// ignore effects
		} else if err2 != nil {
			err = errors.Join(err, fmt.Errorf(
				"contravariant: invalid effect at index %v: %w", i, err2))
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
