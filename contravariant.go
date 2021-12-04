package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// ContravariantPolicy defines related input values.
type ContravariantPolicy struct {
	FilteredScope
}

func (p *ContravariantPolicy) IsVariantKey(
	key interface{},
) (variant bool, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, typ == _interfaceType
	}
	return false, false
}

func (p *ContravariantPolicy) MatchesKey(
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
			return kt.AssignableTo(bt), false
		}
	}
	return false, false
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
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), otherBinding.Strict())
	return !exact && matches
}

func (p *ContravariantPolicy) AcceptResults(
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

func (p *ContravariantPolicy) NewMethodBinding(
	method  reflect.Method,
	spec   *policySpec,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn() - 1  // skip receiver
	args       := make([]arg, numArgs)
	args[0]     = spec.arg
	key        := spec.key
	index      := 1

	// Callback argument must be present if spec
	if len(args) > 1 {
		if arg := methodType.In(2); arg.AssignableTo(_callbackType) {
			args[1] = rawCallbackArg{}
		} else {
			if key == nil { key = arg }
			args[1] = callbackArg{}
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		invalid = errors.New("contravariant: missing callback argument")
	} else if key == nil {
		key = _interfaceType
	}

	if err := buildDependencies(methodType, index, numArgs, args, index); err != nil {
		invalid = multierror.Append(invalid, fmt.Errorf("contravariant: %w", err))
	}

	switch methodType.NumOut() {
	case 0, 1: break
	case 2:
		switch methodType.Out(1) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid, fmt.Errorf(
				"contravariant: when two return values, second must be %v or %v",
				_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid, fmt.Errorf(
			"contravariant: at most two return values allowed and second must be %v or %v",
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
