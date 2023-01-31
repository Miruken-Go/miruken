package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// DiKey represents a key with input and output parts.
	DiKey struct {
		In  any
		Out any
	}

	// BivariantPolicy matches related input and output values.
	BivariantPolicy struct {
		FilteredScope
		in  ContravariantPolicy
		out CovariantPolicy
	}
)

func (p *BivariantPolicy) IsVariantKey(
	key any,
) (variant bool, unknown bool) {
	_, ok := key.(DiKey)
	return ok, false
}

func (p *BivariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches bool, exact bool) {
	if bk, valid := key.(DiKey); valid {
		if ok, valid := otherKey.(DiKey); valid {
			if bk == ok {
				return true, true
			} else if invariant {
				return false, false
			}
			if matches, _ = p.in.MatchesKey(bk.In, ok.In, false); matches {
				matches, _ = p.out.MatchesKey(bk.Out, ok.Out, false)
			}
			return matches, false
		} else {
			panic("expected DiKey for otherBinding.Key()")
		}
	} else {
		panic("expected DiKey for binding.Key()")
	}
}


func (p *BivariantPolicy) Strict() bool {
	return true
}

func (p *BivariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be be nil")
	}
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), false)
	return !exact && matches
}

func (p *BivariantPolicy) AcceptResults(
	results []any,
) (any, HandleResult) {
	return p.out.AcceptResults(results)
}

func (p *BivariantPolicy) NewMethodBinding(
	method reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, key, err := validateBivariantFunc(method.Type, spec, key,1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &MethodBinding{
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags,
				spec.metadata,
			},
			key, method, args, spec.lt,
		}, nil
	}
}

func (p *BivariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, key, err := validateBivariantFunc(fun.Type(), spec, key,0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &FuncBinding{
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags,
				spec.metadata,
			},
			key, fun, args, spec.lt,
		}, nil
	}
}

func validateBivariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, dk any, err error) {
	numArgs := funType.NumIn()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg
	dk       = key
	in      := anyType
	out     := anyType
	index   := 1

	var inv error

	// Callback argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1+skip); arg.AssignableTo(callbackType) {
			args[1] = CallbackArg{}
		} else {
			args[1] = sourceArg{}
			in      = arg
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		err = ErrBiMissingCallback
	}

	if inv := buildDependencies(funType, index+skip, numArgs, args, index); inv != nil {
		err = multierror.Append(err, fmt.Errorf("bivariant: %w", inv))
	}

	switch funType.NumOut() {
	case 0:
		err = multierror.Append(err, ErrBiMissingReturn)
	case 1:
		if out, inv = validateBivariantReturn(funType.Out(0), spec); inv != nil {
			err = multierror.Append(err, inv)
		}
	case 2:
		if out, inv = validateBivariantReturn(funType.Out(0), spec); inv != nil {
			err = multierror.Append(err, inv)
		}
		switch funType.Out(1) {
		case errorType, handleResType: break
		default:
			err = multierror.Append(err, fmt.Errorf(
				"bivariant: when two return values, second must be %v or %v",
				errorType, handleResType))
		}
	default:
		err = multierror.Append(err, fmt.Errorf(
			"bivariant: at most two return values allowed and second must be %v or %v",
			errorType, handleResType))
	}

	if err != nil {
		return nil, dk, err
	}

	if dk == nil {
		dk = DiKey{ In: in, Out: out }
	} else if _, ok := dk.(DiKey); !ok {
		dk = DiKey{ In: dk, Out: out }
	}
	return
}

func validateBivariantReturn(
	returnType reflect.Type,
	spec       *bindingSpec,
) (reflect.Type, error) {
	switch returnType {
	case errorType, handleResType:
		return nil, fmt.Errorf(
			"bivariant: primary return value must not be %v or %v",
			errorType, handleResType)
	default:
		if lt, ok := promise.Inspect(returnType); ok {
			spec.flags = spec.flags | bindingAsync
			returnType = lt
		}
		spec.setLogicalOutputType(returnType)
		return returnType, nil
	}
}

var (
	ErrBiMissingCallback = errors.New("bivariant: missing callback argument")
	ErrBiMissingReturn   = errors.New("bivariant: must have a return value")
)