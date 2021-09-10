package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"strings"
)

// Binding is the abstraction for constraint handling.
type Binding interface {
	Filtered
	Strict()      bool
	SkipFilters() bool
	Constraint()  interface{}

	Matches(
		constraint interface{},
		variance   Variance,
	) (matched bool)

	Invoke(
		context  HandleContext,
		explicitArgs ... interface{},
	) (results []interface{}, err error)
}

type OrderBinding interface {
	Less(binding, otherBinding Binding) bool
}

// MethodBindingError reports a failed method binding.
type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

func (e MethodBindingError) Unwrap() error { return e.Reason }

// methodBinder creates a binding to the `method`
type methodBinder interface {
	newMethodBinding(
		method  reflect.Method,
		spec   *policySpec,
	) (binding Binding, invalid error)
}

// methodInvoke abstracts the invocation of a `method`.
type methodInvoke struct {
	method reflect.Method
	args   []arg
}

func (m methodInvoke) Invoke(
	context      HandleContext,
	explicitArgs ... interface{},
) ([]interface{}, error) {
	fromIndex := len(explicitArgs)
	if args, err := m.resolveArgs(fromIndex, m.args, context); err != nil {
		return nil, err
	} else {
		var values []reflect.Value
		for _, arg := range explicitArgs {
			values = append(values, reflect.ValueOf(arg))
		}
		res := m.method.Func.Call(append(values, args...))
		results := make([]interface{}, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results, nil
	}
}

func (m methodInvoke) resolveArgs(
	fromIndex int,
	args      []arg,
	context   HandleContext,
) ([]reflect.Value, error) {
	var resolved []reflect.Value
	for i, arg := range args {
		typ := m.method.Type.In(fromIndex + i)
		if a, err := arg.resolve(typ, context); err != nil {
			return nil, MethodBindingError{m.method, err}
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
}

// methodBinding models a `constraint` Binding to a method.
type methodBinding struct {
	methodInvoke
	FilteredScope
	constraint interface{}
	flags      bindingFlags
}

func (b *methodBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *methodBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *methodBinding) Constraint() interface{} {
	return b.constraint
}

func (b *methodBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	bc := b.Constraint()
	if bc == constraint {
		return true
	} else if b.Strict() {
		return false
	}
	switch ct := constraint.(type) {
	case reflect.Type:
		if bt, ok := bc.(reflect.Type); ok {
			switch variance {
			case Covariant:
				return bt == _interfaceType || bt.AssignableTo(ct)
			case Contravariant:
				return ct.AssignableTo(bt)
			}
		}
	}
	return false
}

// constructorBinder creates a constructor binding to `handlerType`.
type constructorBinder interface {
	newConstructorBinding(
		handlerType  reflect.Type,
		constructor *reflect.Method,
		spec        *policySpec,
	) (binding Binding, invalid error)
}

// constructorBinding models the creation/initialization
// of the `handlerType`.
type constructorBinding struct {
	FilteredScope
	handlerType  reflect.Type
	flags        bindingFlags
}

func (b *constructorBinding) Strict() bool {
	return false
}

func (b *constructorBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *constructorBinding) Constraint() interface{} {
	return b.handlerType
}

func (b *constructorBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	if variance == Contravariant {
		return false
	}
	if constraint == b.handlerType {
		return true
	}
	if ct, ok := constraint.(reflect.Type); ok {
		return b.handlerType.AssignableTo(ct)
	}
	return false
}

func (b *constructorBinding) Invoke(
	context      HandleContext,
	explicitArgs ... interface{},
) ([]interface{}, error) {
	if len(explicitArgs) > 0 {
		panic("explicitArgs must be empty")
	}
	var handler interface{}
	handlerType := b.handlerType
	if handlerType.Kind() == reflect.Ptr {
		handler = reflect.New(handlerType.Elem()).Interface()
	} else {
		handler = reflect.New(handlerType).Elem().Interface()
	}
	return []interface{}{handler}, nil
}

func newConstructorBinding(
	handlerType  reflect.Type,
	constructor *reflect.Method,
	spec        *policySpec,
) (binding *constructorBinding, invalid error) {
	binding = &constructorBinding{
		handlerType: handlerType,
	}
	if spec != nil {
		binding.providers = spec.filters
		binding.flags     = spec.flags
	}
	if constructor != nil {
		startIndex := 0
		methodType := constructor.Type
		numArgs    := methodType.NumIn() - 1 // skip receiver
		args       := make([]arg, numArgs)
		if spec != nil {
			startIndex = 1
			args[0] = zeroArg{}  // policy/binding placeholder
		}
		for i := startIndex; i < numArgs; i++ {
			if arg, err := buildDependency(methodType.In(i + 1)); err == nil {
				args[i] = arg
			} else {
				invalid = multierror.Append(invalid, fmt.Errorf(
					"constructor: invalid dependency at index %v: %w", i, err))
			}
		}
		if invalid != nil {
			return nil, MethodBindingError{*constructor, invalid}
		}
		initializer := &initializer{methodInvoke{*constructor, args}}
		binding.AddFilters(&initializerProvider{[]Filter{initializer}})
	}
	return binding, nil
}

// Binding builders

type bindingFlags uint8

const (
	bindingNone bindingFlags = 0
	bindingStrict = 1 << iota
	bindingOptional
	bindingSkipFilters
)

type bindingBuilder interface {
	configure(
		index   int,
		field   reflect.StructField,
		binding interface{},
	) error
}

type bindingBuilderFunc func (
	index   int,
	field   reflect.StructField,
	binding interface{},
) error

func (b bindingBuilderFunc) configure(
	index   int,
	field   reflect.StructField,
	binding interface{},
) error {
	return b(index, field, binding)
}

func configureBinding(
	source   reflect.Type,
	binding  interface{},
	builders []bindingBuilder,
) (err error) {
	for i := 0; i < source.NumField(); i++ {
		field := source.Field(i)
		for _, builder := range builders {
			if invalid := builder.configure(i, field, binding); invalid != nil {
				err = multierror.Append(err, invalid)
			}
		}
	}
	return err
}

func optionsBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (err error) {
	if  b, ok := field.Tag.Lookup(_bindingTag); ok {
		if o, ok := binding.(interface {
			bindingAt(int, reflect.StructField) error
		}); ok {
			if invalid := o.bindingAt(index, field); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
		options := strings.Split(b, ",")
		for _, opt := range options {
			switch opt {
			case "": break
			case _strictBinding:
				if b, ok := binding.(interface {
					setStrict(int, reflect.StructField, bool) error
				}); ok {
					if invalid := b.setStrict(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: strict binding on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			case _optionalBinding:
				if b, ok := binding.(interface {
					setOptional(int, reflect.StructField, bool) error
				}); ok {
					if invalid := b.setOptional(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: optional binding on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			case _skipFiltersBinding:
				if b, ok := binding.(interface {
					setSkipFilters(int, reflect.StructField, bool) error
				}); ok {
					if invalid := b.setSkipFilters(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: skipFilters binding on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			default:
				err = multierror.Append(err, fmt.Errorf(
					"binding: invalid binding %q on field %v (%v) of type %T",
					opt, field.Name, index, reflect.TypeOf(binding)))
			}
		}
	}
	return err
}

var (
	_bindingTag         = "bind"
	_strictBinding      = "strict"
	_optionalBinding    = "optional"
	_skipFiltersBinding = "skipFilters"
)