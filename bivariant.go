package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// bivariantPolicy

type bivariantPolicy struct {
	FilteredScope
	input contravariantPolicy
	output covariantPolicy
}

func (p *bivariantPolicy) Variance() Variance {
	return Bivariant
}

func (p *bivariantPolicy) AcceptResults(
	results []interface{},
) (result interface{}, accepted HandleResult) {
	return p.output.AcceptResults(results)
}

func (p *bivariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	key := binding.Key()
	if otherBinding.Matches(key, Invariant) {
		return false
	} else if otherBinding.Matches(key, Covariant) {
		return true
	}
	return false
}

func (p *bivariantPolicy) newMethodBinding(
	method  reflect.Method,
	spec   *policySpec,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn() - 1 // skip receiver
	args       := make([]arg, numArgs)
	args[0]     = zeroArg{}  // policy/binding placeholder

	if err := buildDependencies(methodType, 1, numArgs, args, 1); err != nil {
		invalid = fmt.Errorf("covariant: %w", err)
	}

	switch methodType.NumOut() {
	case 0:
		invalid = multierror.Append(invalid,
			errors.New("covariant: must have a return value"))
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
				"covariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid, fmt.Errorf(
			"covariant: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
	}

	if invalid != nil {
		return nil, MethodBindingError{method, invalid}
	}

	return &methodBinding{
		methodInvoke{method, args},
		FilteredScope{spec.filters},
		spec.key,
		spec.flags,
	}, nil
}

