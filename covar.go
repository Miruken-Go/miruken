package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// CovariantPolicy defines related output values.
type CovariantPolicy struct {
	FilteredScope
}

func (p *CovariantPolicy) IsVariantKey(
	key any,
) (variant bool, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, typ == _anyType
	}
	return false, false
}

func (p *CovariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches bool, exact bool) {
	if key == otherKey {
		return true, true
	} else if invariant {
		return false, false
	}
	switch kt := otherKey.(type) {
	case reflect.Type:
		if bt, isType := key.(reflect.Type); isType {
			return bt == _anyType || bt.AssignableTo(kt), false
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
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), false)
	return !exact && matches
}

func (p *CovariantPolicy) AcceptResults(
	results []any,
) (result any, accepted HandleResult) {
	switch len(results) {
	case 0:
		if IsNil(results) {
			return nil, NotHandled
		}
		return nil, Handled
	case 1:
		if result = results[0]; IsNil(result) {
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
			if IsNil(result) {
				return nil, err.And(NotHandled)
			}
			return result, err
		default:
			if IsNil(result) {
				return nil, NotHandled
			}
			return result, Handled
		}
	}
	return nil, NotHandled.WithError(ErrCovResultsExceeded)
}

func (p *CovariantPolicy) NewMethodBinding(
	method reflect.Method,
	spec   *policySpec,
) (Binding, error) {
	if args, err := validateCovariantFunc(method.Type, spec, 1); err != nil {
		return nil, MethodBindingError{method, err}
	} else {
		return &methodBinding{
			FilteredScope{spec.filters},
			spec.key,
			spec.flags,
			method,
			args,
			spec.metadata,
		}, nil
	}
}

func (p *CovariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *policySpec,
) (Binding, error) {
	if args, err := validateCovariantFunc(fun.Type(), spec, 0); err != nil {
		return nil, FuncBindingError{fun, err}
	} else {
		return &funcBinding{
			FilteredScope{spec.filters},
			spec.key,
			spec.flags,
			fun,
			args,
			spec.metadata,
		}, nil
	}
}

func validateCovariantFunc(
	funType reflect.Type,
	spec    *policySpec,
	skip    int,
) (args []arg, err error) {
	numArgs := funType.NumIn()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg

	if err = buildDependencies(funType, skip+1, numArgs, args, 1); err != nil {
		err = fmt.Errorf("covariant: %w", err)
	}

	switch funType.NumOut() {
	case 0:
		err = multierror.Append(err, ErrCovMissingReturn)
	case 1:
		if inv := validateCovariantReturn(funType.Out(0), spec); inv != nil {
			err = multierror.Append(err, inv)
		}
	case 2:
		if inv := validateCovariantReturn(funType.Out(0), spec); inv != nil {
			err = multierror.Append(err, inv)
		}
		switch funType.Out(1) {
		case _errorType, _handleResType: break
		default:
			err = multierror.Append(err, fmt.Errorf(
				"covariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		err = multierror.Append(err, fmt.Errorf(
			"covariant: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
	}
	return
}

func validateCovariantReturn(
	returnType  reflect.Type,
	spec       *policySpec,
) error {
	switch returnType {
	case _errorType, _handleResType:
		return fmt.Errorf(
			"covariant: primary return value must not be %v or %v",
			_errorType, _handleResType)
	default:
		if spec.key == nil {
			if lt, ok := promise.Inspect(returnType); ok {
				spec.flags = spec.flags | bindingPromise
				returnType = lt
			}
			if spec.flags & bindingStrict != bindingStrict {
				switch returnType.Kind() {
				case reflect.Slice, reflect.Array:
					returnType = returnType.Elem()
				}
			}
			spec.key = returnType
		}
		return nil
	}
}

var (
	ErrCovResultsExceeded = errors.New("covariant: cannot accept more than 2 results")
	ErrCovMissingReturn   = errors.New("covariant: must have a return value")
)