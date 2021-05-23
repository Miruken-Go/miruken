package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"unicode"
)

// arg

type arg interface {
	resolve(
		typ         reflect.Type,
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		composer    Handler,
	) (reflect.Value, error)
}

// receiverArg

type receiverArg struct {}

func (a receiverArg) resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) (reflect.Value, error) {
	return reflect.ValueOf(receiver), nil
}

// zeroArg

type zeroArg struct {}

func (a zeroArg) resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) (reflect.Value, error) {
	return reflect.Zero(typ), nil
}

// callbackArg

type callbackArg struct {}

func (a callbackArg) resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) (reflect.Value, error) {
	if v := reflect.ValueOf(callback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	if v := reflect.ValueOf(rawCallback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("arg: unable to resolve callback: %v", typ)
}

// dependencySpec

type dependencySpec struct {
	index    int
	flags    bindingFlags
	resolver DependencyResolver
}

func (s *dependencySpec) bindingAt(
	index  int,
	field  reflect.StructField,
) error {
	if s.index >= 0 {
		return fmt.Errorf(
			"field index %v already designated as the dependency", s.index)
	}
	if unicode.IsLower(rune(field.Name[0])) {
		return fmt.Errorf(
			"field %v at index %v must start with an uppercase character",
			field.Name, index)
	}
	s.index = index
	return nil
}

func (s *dependencySpec) setStrict(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.flags = s.flags | bindingStrict
	return nil
}

func (s *dependencySpec) setOptional(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.flags = s.flags | bindingOptional
	return nil
}

func (s *dependencySpec) setResolver(
	resolver DependencyResolver,
) error {
	if s.resolver != nil {
		return fmt.Errorf(
			"only one dependency resolver allowed, found %#v", resolver)
	}
	s.resolver = resolver
	return nil
}

// dependencyArg

type dependencyArg struct {
	spec *dependencySpec
}

func (d *dependencyArg) resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	composer    Handler,
) (reflect.Value, error) {
	if typ == _handlerType {
		return reflect.ValueOf(composer), nil
	}
	if val := reflect.ValueOf(rawCallback); val.Type().AssignableTo(typ) {
		return val, nil
	}
	argIndex := -1
	var resolver DependencyResolver = &_defaultArgResolver
	if spec := d.spec; spec != nil {
		argIndex = spec.index
		if spec.resolver != nil {
			resolver = spec.resolver
		}
	}
	val, err := resolver.Resolve(typ, rawCallback, d, composer)
	if err == nil {
		if argIndex >= 0 {
			wrapper := reflect.New(typ.Elem())
			wrapper.Elem().Field(argIndex).Set(val)
			return wrapper, nil
		}
	}
	return val, err
}

// DependencyResolver

type DependencyResolver interface {
	Resolve(
		typ          reflect.Type,
		rawCallback  interface{},
		dep         *dependencyArg,
		handler      Handler,
	) (reflect.Value, error)
}

// defaultDependencyResolver

type defaultDependencyResolver struct{}

func (r *defaultDependencyResolver) Resolve(
	typ          reflect.Type,
	rawCallback  interface{},
	dep         *dependencyArg,
	handler      Handler,
) (reflect.Value, error) {
	argType  := typ
	argIndex := -1
	optional, strict := false, false

	if spec := dep.spec; spec != nil {
		argIndex = spec.index
		optional = spec.flags &bindingOptional == bindingOptional
		strict   = spec.flags &bindingStrict == bindingStrict
		argType  = typ.Elem().Field(argIndex).Type
	}

	var inquiry *Inquiry
	parent, _ := rawCallback.(*Inquiry)

	if !strict && argType.Kind() == reflect.Slice {
		inquiry = NewInquiry(argType.Elem(), true, parent)
	} else {
		inquiry = NewInquiry(argType, false, parent)
	}

	if result, err := inquiry.Resolve(handler); err == nil {
		var val reflect.Value
		if inquiry.Many() {
			results := result.([]interface{})
			val = reflect.New(argType).Elem()
			CopySliceIndirect(results, val)
		} else if result != nil {
			val = reflect.ValueOf(result)
		} else if optional {
			val = reflect.Zero(argType)
		} else {
			return reflect.ValueOf(nil), fmt.Errorf(
				"arg: unable to resolve dependency %v", argType)
		}
		return val, nil
	} else {
		return reflect.ValueOf(nil), fmt.Errorf(
			"arg: unable to resolve dependency %v: %w", argType, err)
	}
}

var dependencyBindingBuilders = []bindingBuilder{
	bindingBuilderFunc(optionsBindingBuilder),
	bindingBuilderFunc(resolverBindingBuilder),
}

func inferDependencyArg(
	argType reflect.Type,
) (*dependencyArg, error) {
	// Is it a *Struct arg specification?
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
		if argType.Kind() == reflect.Struct && argType.Name() == "" {
			spec := &dependencySpec{index: -1}
			if err := configureBinding(argType, spec, dependencyBindingBuilders); err == nil {
				if spec.index < 0 {
					return &_dependencyArg, nil
				}
				dep := &dependencyArg{spec}
				if resolver := spec.resolver; resolver != nil {
					if v, ok := resolver.(interface {
						Validate(reflect.Type, *dependencyArg) error
					}); ok {
						if err := v.Validate(argType, dep); err != nil {
							return nil, err
						}
					}
				}
				return dep, nil
			} else {
				return nil, err
			}
		}
	}
	return &_dependencyArg, nil
}

func resolverBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (err error) {
	if field.Type.AssignableTo(_depArgResolverType) {
		if o, ok := binding.(interface {
			setResolver(resolver DependencyResolver) error
		}); ok {
			resolver := reflect.New(field.Type).Interface().(DependencyResolver)
			if invalid := o.setResolver(resolver); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: dependency resolver %#v at field %v (%v) failed: %w",
					resolver, field.Name, index, invalid))
			}
		}
	}
	return err
}

var (
	_zeroArg            = zeroArg{}
	_callbackArg        = callbackArg{}
	_receiverArg        = receiverArg{}
	_dependencyArg      = dependencyArg{}
	_depArgResolverType = reflect.TypeOf((*DependencyResolver)(nil)).Elem()
	_defaultArgResolver = defaultDependencyResolver{}
)
