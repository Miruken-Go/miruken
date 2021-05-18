package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// covariantPolicy

type covariantPolicy struct{}

func (p *covariantPolicy) Variance() Variance {
	return Covariant
}

func (p *covariantPolicy) AcceptResults(
	results []interface{},
) (result interface{}, accepted HandleResult) {
	return nil, NotHandled
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
	spec   *bindingSpec,

) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn()
	args       := make([]arg, numArgs)

	args[0] = _receiverArg
	args[1] = _zeroArg  // binding placeholder

	for i := 2; i < numArgs; i++ {
		args[i] = dependencyArg{}
	}

	switch methodType.NumOut() {
	case 0:
		invalid = multierror.Append(invalid,
			errors.New("covariant policy: must have a return value"))
	case 1:
		switch methodType.Out(0) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid,
				fmt.Errorf("covariant policy: single return value must not be %v or %v",
					_errorType, _handleResType))
		}
	case 2:
		switch methodType.Out(0) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid,
				fmt.Errorf("covariant policy: when two return values, first must not be %v or %v",
					_errorType, _handleResType))
		}
		switch methodType.Out(1) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid,
				fmt.Errorf("covariant policy: when two return values, second must be %v or %v",
					_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid,
			fmt.Errorf("covariant policy: at most two return values allowed and second must be %v or %v",
				_errorType, _handleResType))
	}

	if invalid != nil {
		return nil, &MethodBindingError{method, invalid}
	}

	return &methodBinding{
		spec:   spec,
		method: method,
		args:   args,
	}, nil
}

