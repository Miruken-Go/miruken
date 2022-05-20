package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

type (
	// DiKey represents a key with input and output parts.
	DiKey struct {
		In  any
		Out any
	}

	// BivariantPolicy defines related input and output values.
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
	strict        bool,
) (matches bool, exact bool) {
	if bk, valid := key.(DiKey); valid {
		if ok, valid := otherKey.(DiKey); valid {
			if bk == ok {
				return true, true
			} else if strict {
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

func (p *BivariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be be nil")
	}
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), otherBinding.Strict())
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
) (Binding, error) {
	if args, key, err := validateBivariantFunc(method.Type, spec, 1); err != nil {
		return nil, MethodBindingError{method, err}
	} else {
		return &methodBinding{
			FilteredScope{spec.filters},
			key,
			spec.flags,
			method,
			args,
		}, nil
	}
}

func (p *BivariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *policySpec,
) (Binding, error) {
	if args, key, err := validateBivariantFunc(fun.Type(), spec, 0); err != nil {
		return nil, FuncBindingError{fun, err}
	} else {
		return &funcBinding{
			FilteredScope{spec.filters},
			key,
			spec.flags,
			fun,
			args,
		}, nil
	}
}

func validateBivariantFunc(
	funType reflect.Type,
	spec    *policySpec,
	skip    int,
) (args []arg, key any, invalid error) {
	numArgs := funType.NumIn()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg
	key      = spec.key
	in      := _anyType
	out     := _anyType
	index   := 1

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
		invalid = errors.New("bivariant: missing callback argument")
	}

	if err := buildDependencies(funType, index+skip, numArgs, args, index); err != nil {
		invalid = multierror.Append(invalid, fmt.Errorf("bivariant: %w", err))
	}

	switch funType.NumOut() {
	case 0:
		invalid = multierror.Append(invalid,
			errors.New("bivariant: must have a return value"))
	case 1:
		out = funType.Out(0)
		if err := validateBivariantReturn(out); err != nil {
			invalid = multierror.Append(invalid, err)
		}
	case 2:
		out = funType.Out(0)
		if err := validateBivariantReturn(out); err != nil {
			invalid = multierror.Append(invalid, err)
		}
		switch funType.Out(1) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid, fmt.Errorf(
				"bivariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid, fmt.Errorf(
			"bivariant: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
	}

	if invalid != nil {
		return nil, key, invalid
	}

	if key == nil {
		key = DiKey{ In: in, Out: out }
	}
	return
}

func validateBivariantReturn(
	returnType reflect.Type,
) error {
	switch returnType {
	case _errorType, _handleResType:
		return fmt.Errorf(
			"bivariant: primary return value must not be %v or %v",
			_errorType, _handleResType)
	default:
		return nil
	}
}
