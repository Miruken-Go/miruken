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
	spec   *policySpec,
	key    any,
) (Binding, error) {
	if args, key, err := validateBivariantFunc(method.Type, spec, key,1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &MethodBinding{
			FilteredScope: FilteredScope{spec.filters},
			key:           key,
			flags:         spec.flags,
			method:        method,
			args:          args,
			metadata:      spec.metadata,
		}, nil
	}
}

func (p *BivariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *policySpec,
	key  any,
) (Binding, error) {
	if args, key, err := validateBivariantFunc(fun.Type(), spec, key,0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &FuncBinding{
			FilteredScope: FilteredScope{spec.filters},
			key:           key,
			flags:         spec.flags,
			fun:           fun,
			args:          args,
			metadata:      spec.metadata,
		}, nil
	}
}

func validateBivariantFunc(
	funType reflect.Type,
	spec    *policySpec,
	key     any,
	skip    int,
) (args []arg, dk any, err error) {
	numArgs := funType.NumIn()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg
	dk       = key
	in      := _anyType
	out     := _anyType
	index   := 1

	var inv error

	// Callback argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1+skip); arg.AssignableTo(_callbackType) {
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
		case _errorType, _handleResType: break
		default:
			err = multierror.Append(err, fmt.Errorf(
				"bivariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		err = multierror.Append(err, fmt.Errorf(
			"bivariant: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
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
	spec       *policySpec,
) (reflect.Type, error) {
	switch returnType {
	case _errorType, _handleResType:
		return nil, fmt.Errorf(
			"bivariant: primary return value must not be %v or %v",
			_errorType, _handleResType)
	default:
		if lt, ok := promise.Inspect(returnType); ok {
			spec.flags = spec.flags | bindingPromise
			return lt, nil
		}
		return returnType, nil
	}
}

var (
	ErrBiMissingCallback = errors.New("bivariant: missing callback argument")
	ErrBiMissingReturn   = errors.New("bivariant: must have a return value")
)