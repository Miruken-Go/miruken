package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// arg models a parameter of a method.
type arg interface {
	resolve(
		typ     reflect.Type,
		context HandleContext,
	) (reflect.Value, error)
}

// zeroArg returns the Zero value of the argument type.
type zeroArg struct {}

func (a zeroArg) resolve(
	typ     reflect.Type,
	context HandleContext,
) (reflect.Value, error) {
	return reflect.Zero(typ), nil
}

// callbackArg returns the callback or raw callback.
type callbackArg struct {}

func (a callbackArg) resolve(
	typ     reflect.Type,
	context HandleContext,
) (reflect.Value, error) {
	if v := reflect.ValueOf(context.Callback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	if v := reflect.ValueOf(context.RawCallback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("arg: unable to resolve callback: %v", typ)
}

// dependencySpec encapsulates dependency metadata.
type dependencySpec struct {
	flags       bindingFlags
	resolver    DependencyResolver
	constraints []func(*ConstraintBuilder)
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

func (s *dependencySpec) addConstraint(
	constraint BindingConstraint,
) error {
	s.constraints = append(s.constraints, func(builder *ConstraintBuilder) {
		builder.WithConstraint(constraint)
	})
	return nil
}

func (s *dependencySpec) unknownBinding(
	index int,
	field reflect.StructField,
) error {
	return nil
}

// DependencyArg is a parameter resolved at runtime.
type DependencyArg struct {
	spec *dependencySpec
}

func (d DependencyArg) resolve(
	typ     reflect.Type,
	context HandleContext,
) (reflect.Value, error) {
	composer := context.Composer
	if typ == _handlerType {
		return reflect.ValueOf(composer), nil
	}
	if typ == _handleCtxType {
		return reflect.ValueOf(context), nil
	}
	rawCallback := context.RawCallback
	if rawCallback != nil {
		if rawType := reflect.TypeOf(rawCallback); rawType.AssignableTo(typ) {
			return reflect.ValueOf(rawCallback), nil
		}
	}
	var resolver DependencyResolver = &_defaultResolver
	if spec := d.spec; spec != nil {
		if spec.resolver != nil {
			resolver = spec.resolver
		}
	}
	val, err := resolver.Resolve(typ, rawCallback, d, composer)
	return val, err
}

// DependencyResolver defines how an argument value is retrieved.
type DependencyResolver interface {
	Resolve(
		typ         reflect.Type,
		rawCallback interface{},
		dep         DependencyArg,
		handler     Handler,
	) (reflect.Value, error)
}

// defaultDependencyResolver retrieves the value from the Handler.
type defaultDependencyResolver struct{}

func (r *defaultDependencyResolver) Resolve(
	typ         reflect.Type,
	rawCallback interface{},
	dep         DependencyArg,
	handler     Handler,
) (reflect.Value, error) {
	optional, strict := false, false
	if spec := dep.spec; spec != nil {
		optional = spec.flags & bindingOptional == bindingOptional
		strict   = spec.flags & bindingStrict == bindingStrict
	}
	var inquiry *Inquiry
	parent, _ := rawCallback.(*Inquiry)
	builder := new(InquiryBuilder).WithParent(parent)
	if !strict && typ.Kind() == reflect.Slice {
		builder.WithKey(typ.Elem()).WithMany()
	} else {
		builder.WithKey(typ)
	}
	if spec := dep.spec; spec != nil {
		builder.WithConstraints(spec.constraints...)
	}
	inquiry = builder.NewInquiry()
	if result, err := inquiry.Resolve(handler); err == nil {
		var val reflect.Value
		if inquiry.Many() {
			results := result.([]interface{})
			val = reflect.New(typ).Elem()
			CopySliceIndirect(results, val)
		} else if result != nil {
			val = reflect.ValueOf(result)
		} else if optional {
			val = reflect.Zero(typ)
		} else {
			return reflect.ValueOf(nil), fmt.Errorf(
				"arg: unable to resolve dependency %v", typ)
		}
		return val, nil
	} else {
		return reflect.ValueOf(nil), fmt.Errorf(
			"arg: unable to resolve dependency %v: %w", typ, err)
	}
}

type HandleContext struct {
	Callback    interface{}
	RawCallback interface{}
	Composer    Handler
	Results     ResultReceiver
}

// Dependency typed

var dependencyBuilders = []bindingBuilder{
	bindingBuilderFunc(optionsBindingBuilder),
	bindingBuilderFunc(resolverBindingBuilder),
	bindingBuilderFunc(constraintBindingBuilder),
}

func buildDependency(
	argType reflect.Type,
) (arg DependencyArg, err error) {
	if argType == _interfaceType {
		return arg, fmt.Errorf(
			"type %v cannot be used as a dependency",
			_interfaceType)
	}
	// Is it a *struct arg binding?
	if argType.Kind() != reflect.Ptr {
		return arg, nil
	}
	argType = argType.Elem()
	if argType.Kind() == reflect.Struct &&
		argType.Name() == "" {  // anonymous
		spec := &dependencySpec{}
		if err = configureBinding(argType, spec, dependencyBuilders); err != nil {
			return arg, err
		}
		arg.spec = spec
	}
	return arg, err
}

func buildDependencies(
	methodType reflect.Type,
	startIndex int,
	endIndex   int,
	args       []arg,
	offset     int,
) (invalid error) {
	var lastSpec *dependencySpec
	for i, j := startIndex, 0; i < endIndex; i, j = i + 1, j + 1 {
		argType := methodType.In(i + 1)  // skip receiver
		if arg, err := buildDependency(argType); err == nil {
			if arg.spec != nil {
				if lastSpec != nil {
					invalid = multierror.Append(invalid, fmt.Errorf(
						"expected dependency at index %v, but found spec", i))
				} else {
					lastSpec = arg.spec // capture spec for actual dependency
					args[j + offset] = zeroArg{}
				}
			} else {
				if lastSpec != nil {
					arg.spec = lastSpec // adopt last dependency spec
					if resolver := lastSpec.resolver; resolver != nil {
						if v, ok := resolver.(interface {
							Validate(reflect.Type, DependencyArg) error
						}); ok {
							if err := v.Validate(argType, arg); err != nil {
								invalid = multierror.Append(invalid, err)
							}
						}
					}
					lastSpec = nil
				}
				args[j + offset] = arg
			}
		} else {
			invalid = multierror.Append(invalid, fmt.Errorf(
				"invalid dependency at index %v: %w", i, err))
		}
	}
	if lastSpec != nil {
		invalid = multierror.Append(invalid, fmt.Errorf(
			"missing dependency at index %v", endIndex))
	}
	return invalid
}

func resolverBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	if dr := coerceToPtr(field.Type, _depResolverType); dr != nil {
		bound = true
		if b, ok := binding.(interface {
			setResolver(resolver DependencyResolver) error
		}); ok {
			if resolver, invalid := newWithTag(dr, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"binding: new dependency resolver at field %v (%v) failed: %w",
					field.Name, index, invalid)
			} else if invalid := b.setResolver(resolver.(DependencyResolver)); invalid != nil {
				err = fmt.Errorf(
					"binding: dependency resolver %#v at field %v (%v) failed: %w",
					resolver, field.Name, index, invalid)
			}
		}
	}
	return bound, err
}

var (
	_handlerType     = reflect.TypeOf((*Handler)(nil)).Elem()
	_handleCtxType   = reflect.TypeOf((*HandleContext)(nil)).Elem()
	_depResolverType = reflect.TypeOf((*DependencyResolver)(nil)).Elem()
	_defaultResolver = defaultDependencyResolver{}
)
