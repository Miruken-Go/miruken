package miruken

import (
	"fmt"
	"reflect"
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
	return fmt.Sprintf("invalid method: %v %v:\n\t* %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

type methodBinder interface {
	newMethodBinding(
		method  reflect.Method,
		spec   *bindingSpec,
	) (binding Binding, invalid error)
}

func (e *MethodBindingError) Unwrap() error { return e.Reason }

// bindingSpec

type bindingSpec struct {
	strict     bool
	constraint interface{}
}

// methodBinding

type methodBinding struct {
	spec   *bindingSpec
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

