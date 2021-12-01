package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// CovariantPolicy defines related output values.
type CovariantPolicy struct {
	FilteredScope
}

func (p *CovariantPolicy) Matches(
	key, otherKey interface{},
	strict        bool,
) (matches bool, exact bool) {
	if key == otherKey {
		return true, true
	} else if strict {
		return false, false
	}
	switch kt := otherKey.(type) {
	case reflect.Type:
		if bt, isType := key.(reflect.Type); isType {
			return bt == _interfaceType || bt.AssignableTo(kt), false
		}
	}
	return false, false
}

func (p *CovariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	matches, exact := p.Matches(otherBinding.Key(), binding.Key(), otherBinding.Strict())
	return !exact && matches
}

func (p *CovariantPolicy) AcceptResults(
	results []interface{},
) (result interface{}, accepted HandleResult) {
	switch len(results) {
	case 0:
		if results == nil {
			return nil, NotHandled
		}
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

func (p *CovariantPolicy) NewMethodBinding(
	method  reflect.Method,
	spec   *policySpec,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn() - 1 // skip receiver
	args       := make([]arg, numArgs)
	args[0]     = spec.arg

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
		if spec.key == nil {
			if spec.flags & bindingStrict != bindingStrict {
				switch returnType.Kind() {
				case reflect.Slice, reflect.Array:
					spec.key = returnType.Elem()
					return nil
				}
			}
			spec.key = returnType
		}
		return nil
	}
}
