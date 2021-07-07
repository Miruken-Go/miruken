package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// contravariantPolicy

type contravariantPolicy struct{
	FilteredScope
}

func (p *contravariantPolicy) Variance() Variance {
	return Contravariant
}

func (p *contravariantPolicy) AcceptResults(
	results []interface{},
) (result interface{}, accepted HandleResult) {
	switch len(results) {
	case 0:
		return nil, Handled
	case 1:
		switch result := results[0].(type) {
		case error:
			return nil, NotHandled.WithError(result)
		case HandleResult:
			return nil, result
		default:
			return result, Handled
		}
	case 2:
		switch result := results[1].(type) {
		case error:
			return results[0], NotHandled.WithError(result)
		case HandleResult:
			return results[0], result
		}
	}
	return nil, NotHandled.WithError(
		errors.New("contravariant policy: cannot accept more than 2 results"))
}

func (p *contravariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be be nil")
	}
	constraint := binding.Constraint()
	if otherBinding.Matches(constraint, Invariant) {
		return false
	} else if otherBinding.Matches(constraint, Contravariant) {
		return true
	}
	return false
}

func (p *contravariantPolicy) newMethodBinding(
	method  reflect.Method,
	spec   *policySpec,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn()
	args       := make([]arg, numArgs)
	constraint := spec.constraint

	args[0] = _receiverArg
	args[1] = _zeroArg  // policy/binding placeholder

	// Callback argument must be present
	if numArgs > 2 {
		if constraint == nil {
			constraint = methodType.In(2)
		}
		args[2] = _callbackArg
	} else {
		invalid = multierror.Append(invalid,
			errors.New("contravariant policy: missing callback argument"))
	}

	for i := 3; i < numArgs; i++ {
		if argType := methodType.In(i); argType == _interfaceType {
			invalid = multierror.Append(invalid, fmt.Errorf(
				"contravariant policy: %v dependency at index %v not allowed",
				_interfaceType, i))
		} else if arg, err := buildDependency(argType); err == nil {
			args[i] = arg
		} else {
			invalid = multierror.Append(invalid, fmt.Errorf(
				"contravariant policy: invalid dependency at index %v: %w", i, err))
		}
	}

	switch methodType.NumOut() {
	case 0, 1: break
	case 2:
		switch methodType.Out(1) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid, fmt.Errorf(
				"contravariant policy: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid, fmt.Errorf(
			"contravariant policy: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
	}

	if invalid != nil {
		return nil, &MethodBindingError{method, invalid}
	}

	return &methodBinding{
		methodInvoke: methodInvoke{method, args},
		constraint:   constraint,
		flags:        spec.flags,
	}, nil
}
