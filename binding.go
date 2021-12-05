package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// Binding abstracts a Callback method.
type Binding interface {
	Filtered
	Key()         interface{}
	Strict()      bool
	SkipFilters() bool
	Invoke(
		ctx HandleContext,
		explicitArgs ... interface{},
	) (results []interface{}, err error)
}

// BindingReducer aggregates Binding results.
type BindingReducer func(
	binding Binding,
	result  HandleResult,
) (HandleResult, bool)

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

// MethodBinder creates a binding to the `method`
type MethodBinder interface {
	NewMethodBinding(
		method  reflect.Method,
		spec   *policySpec,
	) (binding Binding, invalid error)
}

// methodInvoke abstracts the invocation of a `method`.
type methodInvoke struct {
	method reflect.Method
	args   []arg
}

func (m methodInvoke) Method() reflect.Method {
	return m.method
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

// methodBinding models a `key` Binding to a method.
type methodBinding struct {
	methodInvoke
	FilteredScope
	key   interface{}
	flags bindingFlags
}

func (b *methodBinding) Key() interface{} {
	return b.key
}

func (b *methodBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *methodBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

// ConstructorBinder creates a constructor binding to `handlerType`.
type ConstructorBinder interface {
	NewConstructorBinding(
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

func (b *constructorBinding) Key() interface{} {
	return b.handlerType
}

func (b *constructorBinding) Strict() bool {
	return false
}

func (b *constructorBinding) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
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
	bindingStrict bindingFlags = 1 << iota
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

func bindOptions(
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
					"bindOptions: strict field %v (%v) failed: %w",
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
					"bindOptions: optional field %v (%v) failed: %w",
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
					"bindOptions: skipFilters on field %v (%v) failed: %w",
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