package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// DiKey represents a key with input and output parts.
type DiKey struct {
	In  interface{}
	Out interface{}
}

// BivariantPolicy defines related input and output values.
type BivariantPolicy struct {
	CovariantPolicy
}

func (p *BivariantPolicy) Variance() Variance {
	return Bivariant
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
	if key, ok := binding.Key().(DiKey); ok {
		if otherBinding.Matches(key.In, Invariant) {
			if otherBinding.Matches(key.Out, Invariant) {
				return false
			}
			return otherBinding.Matches(key.Out, Covariant)
		}
		return otherBinding.Matches(key.In, Contravariant)
	}
	panic("expected DiKey for BivariantPolicy binding")
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

	// Callback argument must be present
	if len(args) > 1 {
		if key == nil {
			key = methodType.In(2)
		}
		args[1] = callbackArg{}
	} else {
		invalid = errors.New("bivariant: missing callback argument")
	}

	if err := buildDependencies(methodType, 2, numArgs, args, 2); err != nil {
		invalid = multierror.Append(invalid, fmt.Errorf("bivariant: %w", err))
	}

	switch methodType.NumOut() {
	case 0:
		invalid = multierror.Append(invalid,
			errors.New("bivariant: must have a return value"))
	case 1:
		if err := validateCovariantReturn(methodType.Out(0), spec); err != nil {
			invalid = multierror.Append(invalid, err)
		}
	case 2:
		if err := validateCovariantReturn(methodType.Out(0), spec); err != nil {
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

	return &methodBinding{
		methodInvoke{method, args},
		FilteredScope{spec.filters},
		key,
		spec.flags,
	}, nil
}
