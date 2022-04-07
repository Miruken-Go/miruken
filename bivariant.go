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
	return false, false
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
) (result any, accepted HandleResult) {
	return p.out.AcceptResults(results)
}

func (p *BivariantPolicy) NewMethodBinding(
	method  reflect.Method,
	spec   *policySpec,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn() - 1  // skip receiver
	args       := make([]arg, numArgs)
	args[0]     = spec.arg
	key        := spec.key
	in         := _interfaceType
	out        := _interfaceType
	index      := 1

	// Callback argument must be present if spec
	if len(args) > 1 {
		if arg := methodType.In(2); arg.AssignableTo(_callbackType) {
			args[1] = rawCallbackArg{}
		} else {
			args[1] = callbackArg{}
			in      = arg
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		invalid = errors.New("bivariant: missing callback argument")
	}

	if err := buildDependencies(methodType, index, numArgs, args, index); err != nil {
		invalid = multierror.Append(invalid, fmt.Errorf("bivariant: %w", err))
	}

	switch methodType.NumOut() {
	case 0:
		invalid = multierror.Append(invalid,
			errors.New("bivariant: must have a return value"))
	case 1:
		out = methodType.Out(0)
		if err := validateBivariantReturn(out); err != nil {
			invalid = multierror.Append(invalid, err)
		}
	case 2:
		out = methodType.Out(0)
		if err := validateBivariantReturn(out); err != nil {
			invalid = multierror.Append(invalid, err)
		}
		switch methodType.Out(1) {
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
		return nil, MethodBindingError{method, invalid}
	}

	if key == nil {
		key = DiKey{ In: in, Out: out }
	}

	return &methodBinding{
		methodInvoke{method, args},
		FilteredScope{spec.filters},
		key,
		spec.flags,
	}, nil
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
