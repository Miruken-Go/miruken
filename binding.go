package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
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
		return nil, nil  // return nothing if not called as constructor
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
	handlerType   reflect.Type,
	constructor  *reflect.Method,
	spec         *policySpec,
	explicitSpec  bool,
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
		if spec != nil && explicitSpec {
			startIndex = 1
			args[0] = zeroArg{} // policy/binding placeholder
		}
		if err := buildDependencies(methodType, startIndex, numArgs, args, startIndex); err != nil {
			invalid = fmt.Errorf("constructor: %w", err)
		} else {
			initializer := &initializer{methodInvoke{*constructor, args}}
			binding.AddFilters(&initializerProvider{[]Filter{initializer}})
		}
	}
	return binding, invalid
}

// Binding builders

type (
	Strict      struct{}
	Optional    struct{}
	SkipFilters struct{}
)

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
	) (bound bool, err error)
}

type bindingBuilderFunc func (
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error)

func (b bindingBuilderFunc) configure(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	return b(index, field, binding)
}

func configureBinding(
	source   reflect.Type,
	binding  interface{},
	builders []bindingBuilder,
) (err error) {
	for i := 0; i < source.NumField(); i++ {
		bound := false
		field := source.Field(i)
		for _, builder := range builders {
			if b, invalid := builder.configure(i, field, binding); invalid != nil {
				err = multierror.Append(err, invalid)
				break
			} else if b {
				bound = true
				break
			}
		}
		if !bound {
			if b, ok := binding.(interface {
				unknownBinding(int, reflect.StructField) error
			}); ok {
				if invalid := b.unknownBinding(i, field); invalid != nil {
					err = multierror.Append(err, invalid)
				}
			}
		}
	}
	return err
}

func optionsBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	typ := field.Type
	if typ == _strictType {
		bound = true
		if b, ok := binding.(interface {
			setStrict(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setStrict(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: strict binding on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == _optionalType {
		bound = true
		if b, ok := binding.(interface {
			setOptional(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setOptional(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: optional binding on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == _skipFiltersType {
		bound = true
		if b, ok := binding.(interface {
			setSkipFilters(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setSkipFilters(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: skipFilters binding on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	}
	return bound, err
}

var (
	_strictType      = reflect.TypeOf((*Strict)(nil)).Elem()
	_optionalType    = reflect.TypeOf((*Optional)(nil)).Elem()
	_skipFiltersType = reflect.TypeOf((*SkipFilters)(nil)).Elem()
)