package miruken

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// DiKey represents a key with input and output parts.
	DiKey struct {
		In  any
		Out any
	}

	// BivariantPolicy matches related input and output values.
	BivariantPolicy struct {
		FilteredScope
		in  ContravariantPolicy
		out CovariantPolicy
	}
)

var (
	ErrBivMissingCallback = errors.New("bivariant: missing callback argument")
	ErrBivMissingReturn   = errors.New("bivariant: must have a return value")
)

func (p *BivariantPolicy) VariantKey(
	key any,
) (bool, bool) {
	_, ok := key.(DiKey)
	return ok, false
}

func (p *BivariantPolicy) MatchesKey(
	key, otherKey any,
	invariant     bool,
) (matches, exact bool) {
	if bk, valid := key.(DiKey); valid {
		if ok, valid := otherKey.(DiKey); valid {
			if bk == ok {
				return true, true
			} else if invariant {
				return false, false
			}
			if matches, _ = p.in.MatchesKey(bk.In, ok.In, false); matches {
				matches, _ = p.out.MatchesKey(bk.Out, ok.Out, false)
			}
			return matches, false
		} else {
			panic("expected DiKey for otherBinding.Key()")
		}
	} else {
		panic("expected DiKey for binding.Key()")
	}
}

func (p *BivariantPolicy) Strict() bool {
	return true
}

func (p *BivariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be be nil")
	}
	matches, exact := p.MatchesKey(otherBinding.Key(), binding.Key(), false)
	return !exact && matches
}

func (p *BivariantPolicy) AcceptResults(
	results []any,
) (any, HandleResult, []Effect, []any) {
	return p.out.AcceptResults(results)
}

func (p *BivariantPolicy) NewMethodBinding(
	method *reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, k, err := validateBivariantFunc(method.Type, spec, key, 1); err != nil {
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

func (p *BivariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, k, err := validateBivariantFunc(fun.Type(), spec, key, 0); err != nil {
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

func validateBivariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, dk any, err error) {
	numArgs := funType.NumIn()
	numOut := funType.NumOut()
	args = make([]arg, numArgs-skip)
	args[0] = spec.arg
	dk = key
	in := internal.AnyType
	out := internal.AnyType
	index := 1

	// Callback argument must be present if spec
	if len(args) > 1 {
		if arg := funType.In(1 + skip); arg.AssignableTo(callbackType) {
			args[1] = CallbackArg{}
		} else {
			args[1] = sourceArg{}
			in = arg
		}
		index++
	} else if _, isSpec := spec.arg.(zeroArg); isSpec {
		err = ErrBivMissingCallback
	}

	if err2 := buildDependencies(funType, index+skip, numArgs, args, index); err2 != nil {
		err = errors.Join(err, fmt.Errorf("bivariant: %w", err2))
	}

	resIdx := -1

	for i := range numOut {
		oo := funType.Out(i)
		switch {
		case oo.AssignableTo(internal.ErrorType):
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"bivariant: error found at index %v must be last return", i))
			}
		case oo.AssignableTo(handleResType):
			if i != numOut-1 {
				err = errors.Join(err, fmt.Errorf(
					"bivariant: HandleResult found at index %v must be last return", i))
			}
		case resIdx >= 0:
			err = errors.Join(err, fmt.Errorf(
				"bivariant: effective return at index %v conflicts with index %v", i, resIdx))
		default:
			out = oo
			resIdx = i
			if lt, ok := promise.Inspect(out); ok {
				spec.flags |= bindingAsync
				out = lt
			}
			spec.setLogicalOutputType(out)
		}
	}

	if resIdx < 0 {
		err = errors.Join(err, ErrBivMissingReturn)
	}

	if err != nil {
		return nil, dk, err
	} else if dk == nil {
		dk = DiKey{In: in, Out: out}
	} else if _, ok := dk.(DiKey); !ok {
		dk = DiKey{In: dk, Out: out}
	}
	return
}
