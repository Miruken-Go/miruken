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

// methodBinder

type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

type methodBinder interface {
	newMethodBinding(
		method  reflect.Method,
		spec   *methodSpec,
	) (binding Binding, invalid error)
}

func (e *MethodBindingError) Unwrap() error { return e.Reason }

// methodSpec

type methodSpec struct {
	strict     bool
	constraint interface{}
}

func (s *methodSpec) bindingAt(
	index  int,
	field  reflect.StructField,
) error {
	if index != 0 {
		return fmt.Errorf(
			"method binding must be at index 0, found at %v", index)
	}
	return nil
}

func (s *methodSpec) setStrict(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.strict = strict
	return nil
}

// methodBinding

type methodBinding struct {
	spec   *methodSpec
	method  reflect.Method
	args    []arg
}

func (b *methodBinding) Strict() bool {
	return b.spec != nil && b.spec.strict
}

func (b *methodBinding) Constraint() interface{} {
	return b.spec.constraint
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

func (b *methodBinding) Invoke(
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
)  ([]interface{}, error) {
	if args, err := b.resolveArgs(
		b.args, receiver, callback, rawCallback, composer); err != nil {
		return nil, err
	} else {
		res := b.method.Func.Call(args)
		results := make([]interface{}, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results, nil
	}
}

func (b *methodBinding) resolveArgs(
	args        []arg,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) ([]reflect.Value, error) {
	var resolved []reflect.Value
	for i, arg := range args {
		typ := b.method.Type.In(i)
		if a, err := arg.resolve(typ, receiver, callback, rawCallback, composer); err != nil {
			return nil, &MethodBindingError{b.method, err}
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
}

// constructorBinding

type constructorBinding struct {
	handlerType reflect.Type
}

func (b *constructorBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	return false
}

func (b *constructorBinding) Invoke(
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) (results []interface{}) {
	return nil
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
				if o, ok := binding.(interface {
					setStrict(int, reflect.StructField, bool) error
				}); ok {
					if invalid := o.setStrict(index, field, true); invalid != nil {
						err = multierror.Append(err, fmt.Errorf(
							"binding: strict option on field %v (%v) failed: %w",
							field.Name, index, invalid))
					}
				}
			case _optionalOption:
				if o, ok := binding.(interface {
					setOptional(int, reflect.StructField, bool) error
				}); ok {
					if invalid := o.setOptional(index, field, true); invalid != nil {
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