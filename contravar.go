package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// ContravariantPolicy defines related input values.
type ContravariantPolicy struct {
	FilteredScope
}

func (p *ContravariantPolicy) IsVariantKey(
	key any,
) (variant bool, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, typ == _anyType
	}
	return false, false
}

func (p *ContravariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches bool, exact bool) {
	if key == otherKey {
		return true, true
	} else if invariant {
		return false, false
	} else if bt, isType := key.(reflect.Type); isType {
		if bt == _anyType {
			return true, false
		} else if kt, isType := otherKey.(reflect.Type); isType {
			return kt.AssignableTo(bt), false
		}
	}
	return false, false
}

func (p *ContravariantPolicy) Strict() bool {
	return true
}

func (p *ContravariantPolicy) Less(
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

func (p *ContravariantPolicy) AcceptResults(
	results []any,
) (result any, accepted HandleResult) {
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
		default:
			return results[0], Handled
		}
	}
	return nil, NotHandled.WithError(ErrConResultsExceeded)
}

func (p *ContravariantPolicy) NewMethodBinding(
	method reflect.Method,
	spec   *policySpec,
) (Binding, error) {
	if args, key, err := validateContravariantFunc(method.Type, spec, 1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &methodBinding{
			FilteredScope{spec.filters},
			key,
			spec.flags,
			method,
			args,
			spec.metadata,
		}, nil
	}
}

func (p *ContravariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *policySpec,
) (Binding, error) {
	if args, key, err := validateContravariantFunc(fun.Type(), spec, 0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &funcBinding{
			FilteredScope{spec.filters},
			key,
			spec.flags,
			fun,
			args,
			spec.metadata,
		}, nil
	}
}

func validateContravariantFunc(
	funType reflect.Type,
	spec    *policySpec,
	skip    int,
) (args []arg, key any, err error) {
	numArgs := funType.NumIn()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg
	key      = spec.key
	index   := 1

	// Source argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1+skip); arg.AssignableTo(_callbackType) {
			args[1] = CallbackArg{}
			if key == nil {
				key = _anyType
			}
		} else {
			if key == nil {
				key = arg
			}
			args[1] = sourceArg{}
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		err = ErrConMissingCallback
	} else if key == nil {
		key = _anyType
	}

	if inv := buildDependencies(funType, index+skip, numArgs, args, index); inv != nil {
		err = multierror.Append(err, fmt.Errorf("contravariant: %w", inv))
	}

	switch funType.NumOut() {
	case 0: break
	case 1:
		if _, ok := promise.Inspect(funType.Out(0)); ok {
			spec.flags = spec.flags | bindingPromise
		}
	case 2:
		if _, ok := promise.Inspect(funType.Out(0)); ok {
			spec.flags = spec.flags | bindingPromise
		}
		switch funType.Out(1) {
		case _errorType, _handleResType: break
		default:
			err = multierror.Append(err, fmt.Errorf(
				"contravariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		err = multierror.Append(err, fmt.Errorf(
			"contravariant: at most two return values allowed and second must be %v or %v",
			_errorType, _handleResType))
	}
	return
}

var (
	ErrConResultsExceeded = errors.New("contravariant: cannot accept more than 2 results")
	ErrConMissingCallback = errors.New("contravariant: missing callback argument")
)