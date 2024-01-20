package miruken

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

// ContravariantPolicy matches related input values.
type ContravariantPolicy struct {
	FilteredScope
}

var ErrConMissingCallback = errors.New("contravariant: missing callback argument")

func (p *ContravariantPolicy) VariantKey(
	key any,
) (variant, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, internal.IsAny(typ)
	}
	return false, false
}

func (p *ContravariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches, exact bool) {
	if key == otherKey {
		return true, true
	} else if invariant {
		return false, false
	} else if bt, isType := key.(reflect.Type); isType {
		if internal.IsAny(bt) {
			return true, false
		} else if kt, isType := otherKey.(reflect.Type); isType {
			if kt.AssignableTo(bt) {
				return true, false
			}
			if kt.Kind() == reflect.Ptr && kt.Elem().AssignableTo(bt) {
				return true, false
			}
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
) (any, HandleResult, []Intent) {
	switch len(results) {
	case 0:
		return nil, Handled, nil
	case 1:
		switch result := results[0].(type) {
		case error:
			return nil, NotHandled.WithError(result), nil
		case HandleResult:
			return nil, result, nil
		default:
			if intent, _ := MakeIntent(result, false); intent != nil {
				return nil, Handled, []Intent{intent}
			}
			return result, Handled, nil
		}
	default:
		iEnd := 0
		hr := Handled
		switch err := results[len(results)-1].(type) {
		case error:
			return nil, NotHandled.WithError(err), nil
		case HandleResult:
			if !err.Handled() || err.IsError() {
				return nil, err, nil
			}
			hr = err
			iEnd++
		}
		iStart := 0
		var res any
		if h, ok := results[0].(HandleResult); ok {
			if !h.Handled() || h.IsError() {
				return nil, h, nil
			}
			hr = h
			iStart++
		} else {
			if intent, _ := MakeIntent(results[0], false); intent == nil {
				res = results[0]
				iStart++
			}
		}
		intents, err := MakeIntents(true, results[iStart:len(results)-iEnd])
		if err != nil {
			return res, NotHandled.WithError(err), nil
		}
		return res, hr, intents
	}
}

func (p *ContravariantPolicy) NewMethodBinding(
	method *reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, k, err := validateContravariantFunc(method.Type, spec, key, 1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &methodBinding{
			funcCall{method.Func, args},
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags, spec.metadata,
			}, k, *method, spec.lt,
		}, nil
	}
}

func (p *ContravariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, k, err := validateContravariantFunc(fun.Type(), spec, key, 0); err != nil {
		return nil, &FuncBindingError{fun, err}
	} else {
		return &funcBinding{
			funcCall{fun, args},
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags, spec.metadata,
			}, k, spec.lt,
		}, nil
	}
}

func validateContravariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, ck any, err error) {
	ck = key
	numArgs := funType.NumIn()
	numOut := funType.NumOut()
	args = make([]arg, numArgs-skip)
	args[0] = spec.arg
	index := 1

	// Source argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1 + skip); arg.AssignableTo(callbackType) {
			args[1] = CallbackArg{}
			if ck == nil {
				ck = internal.AnyType
			}
		} else {
			if ck == nil {
				ck = arg
			}
			args[1] = sourceArg{}
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		err = ErrConMissingCallback
	} else if ck == nil {
		ck = internal.AnyType
	}

	if err2 := buildDependencies(funType, index+skip, numArgs, args, index); err2 != nil {
		err = multierror.Append(err, fmt.Errorf("contravariant: %w", err2))
	}

	resIdx := -1

	for i := 0; i < numOut; i++ {
		out := funType.Out(i)
		if out.AssignableTo(internal.ErrorType) {
			if i != numOut-1 {
				err = multierror.Append(err, fmt.Errorf(
					"contravariant: error found at index %v must be last return", i))
			}
		} else if out.AssignableTo(handleResType) {
			if i != numOut-1 {
				err = multierror.Append(err, fmt.Errorf(
					"contravariant: HandleResult found at index %v must be last return", i))
			}
		} else if ok, err2 := ValidIntent(out); ok {
			// ignore intents
		} else if err2 != nil {
			err = multierror.Append(err, fmt.Errorf(
				"contravariant: invalid intent at index %v: %w", i, err2))
		} else if resIdx >= 0 {
			err = multierror.Append(err, fmt.Errorf(
				"contravariant: effective return at index %v conflicts with index %v", i, resIdx))
		} else {
			resIdx = i
			if lt, ok := promise.Inspect(out); ok {
				spec.flags |= bindingAsync
				out = lt
			}
			spec.setLogicalOutputType(out)
		}
	}
	return
}
