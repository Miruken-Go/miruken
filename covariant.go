package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// covariantPolicy

type covariantPolicy struct{
	FilteredScope
}

func (p *covariantPolicy) Variance() Variance {
	return Covariant
}

func (p *covariantPolicy) AcceptResults(
	results []interface{},
) (result interface{}, accepted HandleResult) {
	switch len(results) {
	case 0:
		return nil, Handled
	case 1:
		if result = results[0]; result == nil {
			return nil, NotHandled
		} else if r, ok := result.(HandleResult); ok {
			return nil, r
		}
		return result, Handled
	case 2:
		result = results[0]
		switch err := results[1].(type) {
		case error:
			return result, NotHandled.WithError(err)
		case HandleResult:
			if result == nil {
				return nil, err.And(NotHandled)
			}
			return result, err
		default:
			if result == nil {
				return nil, NotHandled
			}
			return result, Handled
		}
	}
	return nil, NotHandled.WithError(
		errors.New("covariant policy: cannot accept more than 2 results"))
}

func (p *covariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	constraint := binding.Constraint()
	if otherBinding.Matches(constraint, Invariant) {
		return false
	} else if otherBinding.Matches(constraint, Covariant) {
		return true
	}
	return false
}

func (p *covariantPolicy) newMethodBinding(
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
		spec.constraint,
		spec.flags,
	}, nil
}

func validateCovariantReturn(
	returnType  reflect.Type,
	spec       *policySpec,
) error {
	switch returnType {
	case _errorType, _handleResType:
		return fmt.Errorf(
			"covariant policy: primary return value must not be %v or %v",
			_errorType, _handleResType)
	default:
		if spec.constraint == nil {
			if spec.flags & bindingStrict != bindingStrict {
				switch returnType.Kind() {
				case reflect.Slice, reflect.Array:
					spec.constraint = returnType.Elem()
					return nil
				}
			}
			spec.constraint = returnType
		}
		return nil
	}
}
