package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/internal"
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
		return true, internal.AnyType.AssignableTo(typ)
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
		if internal.AnyType.AssignableTo(bt) {
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
		if internal.IsNil(results) {
			return nil, NotHandled
		}
		return nil, Handled
	case 1:
		if result = results[0]; internal.IsNil(result) {
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
			if internal.IsNil(result) {
				return nil, err.And(NotHandled)
			}
			return result, err
		default:
			if internal.IsNil(result) {
				return nil, NotHandled
			}
			return result, Handled
		}
	}
	return nil, NotHandled.WithError(ErrCovResultsExceeded)
}

func (p *CovariantPolicy) NewCtorBinding(
	typ   reflect.Type,
	ctor  *reflect.Method,
	inits []reflect.Method,
	spec  *bindingSpec,
	key   any,
) (Binding, error) {
	binding := &ctorBinding{typ: typ, key: key}
	if spec != nil {
		binding.BindingBase.FilteredScope.providers = spec.filters
		binding.BindingBase.metadata = spec.metadata
		binding.BindingBase.flags = spec.flags
	}
	if ctor != nil {
		startIndex := 0
		methodType := ctor.Type
		numArgs    := methodType.NumIn()
		args       := make([]arg, numArgs-1)  // skip receiver
		if spec != nil {
			startIndex = 1
			args[0] = zeroArg{}  // policy/binding placeholder
		}
		err := buildDependencies(methodType, startIndex+1, numArgs, args, startIndex)
		if err != nil {
			return nil, fmt.Errorf("constructor: %w", err)
		}
		initializer := &initializer{ctor:funcCall{ctor.Func, args}}
		binding.AddFilters(&initProvider{[]Filter{initializer}})
	}
	return binding, nil
}

func (p *CovariantPolicy) NewMethodBinding(
	method reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, key, err := validateCovariantFunc(method.Type, spec, key,1); err != nil {
		return nil, &MethodBindingError{method, err}
	} else {
		return &methodBinding{
			funcCall{method.Func, args},
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags, spec.metadata,
			}, key, method, spec.lt,
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
		return &funcBinding{
			funcCall{fun, args},
			BindingBase{
				FilteredScope{spec.filters},
				spec.flags, spec.metadata,
			}, key, spec.lt,
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
	numOut  := funType.NumOut()
	args     = make([]arg, numArgs-skip)
	args[0]  = spec.arg

	if err = buildDependencies(funType, skip+1, numArgs, args, 1); err != nil {
		err = fmt.Errorf("covariant: %w", err)
	}

	resIdx := -1

	for i := 0; i < numOut; i++ {
		out := funType.Out(i)
		if out.AssignableTo(internal.ErrorType) {
			if i != numOut-1 {
				err = multierror.Append(err, fmt.Errorf(
					"covariant: error found at index %v must be last return", i))
			}
		} else if out.AssignableTo(handleResType) {
			if i != numOut-1 {
				err = multierror.Append(err, fmt.Errorf(
					"covariant: HandleResult found at index %v must be last return", i))
			}
		} else if out.AssignableTo(sideEffectType) {
			// ignore side-effects
		} else if resIdx >= 0 {
			err = multierror.Append(err, fmt.Errorf(
				"covariant: effective return at index %v conflicts with index %v", i, resIdx))
		} else {
			resIdx = i
			if key == nil {
				if lt, ok := promise.Inspect(out); ok {
					spec.flags = spec.flags | bindingAsync
					out = lt
				}
				if spec.flags & bindingStrict != bindingStrict {
					switch out.Kind() {
					case reflect.Slice, reflect.Array:
						out = out.Elem()
					}
				}
				key = out
			}
			ck  = key
			spec.setLogicalOutputType(out)
		}
	}

	if resIdx < 0 {
		err = multierror.Append(err, ErrCovMissingReturn)
	}
	return
}
