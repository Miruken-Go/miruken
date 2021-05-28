package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"strings"
)

// Binding

type Binding interface {
	Strict()     bool
	Constraint() interface{}

	Matches(
		constraint interface{},
		variance   Variance,
	) (matched bool)

	Invoke(
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		composer    Handler,
	) (results []interface{}, err error)
}

type OrderBinding interface {
	Less(binding, otherBinding Binding) bool
}

// MethodBindingError

type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

func (e *MethodBindingError) Unwrap() error { return e.Reason }

// methodBinder

type methodBinder interface {
	newMethodBinding(
		method  reflect.Method,
		spec   *policySpec,
	) (binding Binding, invalid error)
}

// methodInvoke

type methodInvoke struct {
	method reflect.Method
	args   []arg
}

func (m *methodInvoke) Invoke(
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
)  ([]interface{}, error) {
	if args, err := m.resolveArgs(
		m.args, receiver, callback, rawCallback, composer); err != nil {
		return nil, err
	} else {
		res := m.method.Func.Call(args)
		results := make([]interface{}, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results, nil
	}
}

func (m *methodInvoke) resolveArgs(
	args        []arg,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) ([]reflect.Value, error) {
	var resolved []reflect.Value
	for i, arg := range args {
		typ := m.method.Type.In(i)
		if a, err := arg.resolve(typ, receiver, callback, rawCallback, composer); err != nil {
			return nil, &MethodBindingError{m.method, err}
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
}

// methodBinding

type methodBinding struct {
	methodInvoke
	constraint interface{}
	flags      bindingFlags
}

func (b *methodBinding) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
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

// constructorBinder

type constructorBinder interface {
	newConstructorBinding(
		handlerType  reflect.Type,
		initMethod  *reflect.Method,
		spec        *policySpec,
	) (binding Binding, invalid error)
}

// constructorBinding

type constructorBinding struct {
	handlerType  reflect.Type
	initMethod  *methodInvoke
}

func (b *constructorBinding) Strict() bool {
	return false
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
	if constraint == _handlerType {
		return true
	}
	if ct, ok := constraint.(reflect.Type); ok {
		return _handlerType.AssignableTo(ct)
	}
	return false
}

func (b *constructorBinding) Invoke(
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) ([]interface{}, error) {
	if receiver != nil {
		panic("receiver must be nil")
	}
	handler := reflect.New(b.handlerType)
	if initMethod := b.initMethod; initMethod != nil {
		if _, err := initMethod.Invoke(
			handler, callback, rawCallback, composer); err != nil {
			return nil, err
		}
	}
	return []interface{}{handler}, nil
}

func newConstructorBinding(
	handlerType  reflect.Type,
	initMethod  *reflect.Method,
	spec        *policySpec,
) (binding *constructorBinding, invalid error) {
	var invokeInit *methodInvoke
	if initMethod != nil {
		startIndex := 1
		methodType := initMethod.Type
		numArgs    := methodType.NumIn()
		args       := make([]arg, numArgs)
		args[0]     = _receiverArg
		if spec != nil {
			startIndex = 2
		}
		for i := startIndex; i < numArgs; i++ {
			if argType := methodType.In(i); argType == _interfaceType {
				invalid = multierror.Append(invalid, fmt.Errorf(
					"init: %v dependency at index %v not allowed",
					_interfaceType, i))
			} else if arg, err := buildDependency(argType); err == nil {
				args[i] = arg
			} else {
				invalid = multierror.Append(invalid, fmt.Errorf(
					"init: invalid dependency at index %v: %w", i, err))
			}
		}

		if invalid != nil {
			return nil, &MethodBindingError{*initMethod, invalid}
		}

		invokeInit = &methodInvoke{*initMethod, args}
	}

	return &constructorBinding{
		handlerType, invokeInit,
	}, nil
}

// Binding builders

type bindingFlags uint8

const (
	bindingNone bindingFlags = 0
	bindingStrict = 1 << iota
	bindingOptional
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
			case _strictOption:
				if b, ok := binding.(interface {
					setStrict(int, reflect.StructField, bool) error
				}); ok {
					if invalid := b.setStrict(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: strict option on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			case _optionalOption:
				if b, ok := binding.(interface {
					setOptional(int, reflect.StructField, bool) error
				}); ok {
					if invalid := b.setOptional(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: optional option on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			default:
				err = multierror.Append(err, fmt.Errorf(
					"binding: invalid option %q on field %v (%v) for binding %T",
					opt, field.Name, index, reflect.TypeOf(binding)))
			}
		}
	}
	return err
}

var (
	_bindingTag     = "bind"
	_strictOption   = "strict"
	_optionalOption = "optional"
)