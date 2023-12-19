package miruken

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
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
) (variant, unknown bool) {
	if typ, ok := key.(reflect.Type); ok {
		return true, internal.IsAny(typ)
	}
	return false, false
}

func (p *CovariantPolicy) MatchesKey(
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
	inits []*reflect.Method,
	spec  *bindingSpec,
	key   any,
) (Binding, error) {
	binding := &ctorBinding{typ: typ, key: key}
	if spec != nil {
		binding.BindingBase.FilteredScope.providers = spec.filters
		binding.BindingBase.metadata = spec.metadata
		binding.BindingBase.flags = spec.flags
	}
	var ctorInits initializer
	if ctor != nil {
		if err := addInitializer(ctor, &ctorInits, spec != nil, "constructor"); err != nil {
			return nil, err
		}
	}
	for _, init := range inits {
		implicit := strings.HasPrefix(init.Name, "Init")
		if err := addInitializer(init, &ctorInits, !implicit, "initializer"); err != nil {
			return nil, err
		}
	}
	if len(ctorInits.inits) > 0 {
		binding.AddFilters(&initProvider{[]Filter{&ctorInits}})
	}
	return binding, nil
}

func (p *CovariantPolicy) NewMethodBinding(
	method *reflect.Method,
	spec   *bindingSpec,
	key    any,
) (Binding, error) {
	if args, k, err := validateCovariantFunc(method.Type, spec, key, 1); err != nil {
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

func (p *CovariantPolicy) NewFuncBinding(
	fun  reflect.Value,
	spec *bindingSpec,
	key  any,
) (Binding, error) {
	if args, k, err := validateCovariantFunc(fun.Type(), spec, key, 0); err != nil {
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

func validateCovariantFunc(
	funType reflect.Type,
	spec    *bindingSpec,
	key     any,
	skip    int,
) (args []arg, ck any, err error) {
	numArgs := funType.NumIn()
	numOut := funType.NumOut()
	args = make([]arg, numArgs-skip)
	args[0] = spec.arg

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
					spec.flags |= bindingAsync
					out = lt
				}
				if spec.flags&bindingStrict != bindingStrict {
					switch out.Kind() {
					case reflect.Slice, reflect.Array:
						out = out.Elem()
					}
				}
				key = out
			}
			ck = key
			spec.setLogicalOutputType(out)
		}
	}

	if resIdx < 0 {
		err = multierror.Append(err, ErrCovMissingReturn)
	}
	return
}

func addInitializer(
	init      *reflect.Method,
	ci        *initializer,
	zeroFirst bool,
	label     string,
) error {
	startIndex := 0
	initType := init.Type
	numArgs := initType.NumIn()
	args := make([]arg, numArgs-1) // skip receiver
	if zeroFirst {
		startIndex = 1
		args[0] = zeroArg{}
	}
	err := buildDependencies(initType, startIndex+1, numArgs, args, startIndex)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	ci.inits = append(ci.inits, funcCall{init.Func, args})
	return nil
}
