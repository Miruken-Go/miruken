package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// CovariantPolicy matches related output values.
type CovariantPolicy struct {
	FilteredScope
}


var (
	ErrCovResultsExceeded = errors.New("covariant: cannot accept more than 2 results")
	ErrCovMissingReturn   = errors.New("covariant: must have a return value")
)


func (p *CovariantPolicy) VariantKey(
	key any,
) (variant bool, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, anyType.AssignableTo(typ)
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
	} else if bt, isType := key.(reflect.Type); isType {
		if anyType.AssignableTo(bt) {
			return true, false
		} else if kt, isType := otherKey.(reflect.Type); isType {
			return bt.AssignableTo(kt), false
		}
	}
	return false, false
}

func (p *CovariantPolicy) Strict() bool {
	return false
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
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, key, err := validateCovariantFunc(method.Type, spec, key,1); err != nil {
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

func (p *CovariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, key, err := validateCovariantFunc(fun.Type(), spec, key,0); err != nil {
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


func validateCovariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, ck any, err error) {
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
		if key, inv := validateCovariantReturn(funType.Out(0), spec, key, false); inv != nil {
			err = multierror.Append(err, inv)
		} else {
			ck = key
		}
	case 2:
		if key, inv := validateCovariantReturn(funType.Out(0), spec, key, true); inv != nil {
			err = multierror.Append(err, inv)
		} else {
			ck = key
		}
		switch funType.Out(1) {
		case errorType, handleResType: break
		default:
			err = multierror.Append(err, fmt.Errorf(
				"covariant: when two return values, second must be %v or %v",
				errorType, handleResType))
		}
	default:
		err = multierror.Append(err, fmt.Errorf(
			"covariant: at most two return values allowed and second must be %v or %v",
			errorType, handleResType))
	}
	return
}

func validateCovariantReturn(
	returnType reflect.Type,
	spec       *bindingSpec,
	key        any,
	allowErr   bool,
) (any, error) {
	switch returnType {
	case errorType:
		if !allowErr {
			return nil, fmt.Errorf(
				"covariant: primary return value must not be %v", errorType)
		}
	case handleResType:
		return nil, fmt.Errorf(
			"covariant: primary return value must not be %v", handleResType)
	}
	if key == nil {
		if lt, ok := promise.Inspect(returnType); ok {
			spec.flags = spec.flags | bindingAsync
			returnType = lt
		}
		if spec.flags & bindingStrict != bindingStrict {
			switch returnType.Kind() {
			case reflect.Slice, reflect.Array:
				returnType = returnType.Elem()
			}
		}
		key = returnType
	}
	spec.setLogicalOutputType(returnType)
	return key, nil
}
