package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// arg models a parameter of a method.
type arg interface {
	resolve(
		typ reflect.Type,
		ctx HandleContext,
	) (reflect.Value, error)
}

// zeroArg returns the Zero value of the argument type.
type zeroArg struct {}

func (a zeroArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	return reflect.Zero(typ), nil
}

// rawCallbackArg returns the raw callback.
type rawCallbackArg struct {}

func (a rawCallbackArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(ctx.RawCallback()), nil
}

// callbackArg returns the raw callback.
type callbackArg struct {}

func (a callbackArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	if v := reflect.ValueOf(ctx.Callback()); v.Type().AssignableTo(typ) {
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

// DependencyArg is a parameter resolved at runtime.
type DependencyArg struct {
	spec *dependencySpec
}

func (d DependencyArg) Optional() bool {
	return d.spec != nil && d.spec.flags & bindingOptional == bindingOptional
}

func (d DependencyArg) Strict() bool {
	return d.spec != nil && d.spec.flags & bindingStrict == bindingStrict
}

func (d DependencyArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	composer := ctx.Composer()
	if typ == _handlerType {
		return reflect.ValueOf(composer), nil
	}
	if typ == _handleCtxType {
		return reflect.ValueOf(ctx), nil
	}
	if val := reflect.ValueOf(ctx.Callback()); val.Type().AssignableTo(typ) {
		return val, nil
	}
	rawCallback := ctx.RawCallback()
	if val := reflect.ValueOf(rawCallback); val.Type().AssignableTo(typ) {
		return val, nil
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
		rawCallback Callback,
		dep         DependencyArg,
		handler     Handler,
	) (reflect.Value, error)
}

// defaultDependencyResolver resolves the value from the Handler.
type defaultDependencyResolver struct{}

func (r *defaultDependencyResolver) Resolve(
	typ         reflect.Type,
	rawCallback Callback,
	dep         DependencyArg,
	handler     Handler,
) (reflect.Value, error) {
	var provides *Provides
	parent, _ := rawCallback.(*Provides)
	var builder ProvidesBuilder
	builder.WithParent(parent)
	if !dep.Strict() && typ.Kind() == reflect.Slice {
		builder.WithKey(typ.Elem()).WithMany()
	} else {
		builder.WithKey(typ)
	}
	if spec := dep.spec; spec != nil {
		builder.WithConstraints(spec.constraints...)
	}
	provides = builder.NewProvides()
	if result, err := provides.Resolve(handler); err == nil {
		var val reflect.Value
		if provides.Many() {
			results := result.([]any)
			val = reflect.New(typ).Elem()
			CopySliceIndirect(results, val)
		} else if result != nil {
			val = reflect.ValueOf(result)
		} else if dep.Optional() {
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
	callback    any
	rawCallback Callback
	binding     Binding
	composer    Handler
}

func (h HandleContext) Callback() any {
	return h.callback
}

func (h HandleContext) RawCallback() Callback {
	return h.rawCallback
}

func (h HandleContext) Binding() Binding {
	return h.binding
}

func (h HandleContext) Composer() Handler {
	return h.composer
}

// Dependency typed

var dependencyBuilders = []bindingBuilder{
	bindingBuilderFunc(bindOptions),
	bindingBuilderFunc(bindResolver),
	bindingBuilderFunc(bindConstraints),
}

func buildDependency(
	argType reflect.Type,
) (arg DependencyArg, err error) {
	if argType == _anyType {
		return arg, fmt.Errorf(
			"type %v cannot be used as a dependency",
			_anyType)
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

func bindResolver(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	if dr := coerceToPtr(field.Type, _depResolverType); dr != nil {
		bound = true
		if b, ok := binding.(interface {
			setResolver(resolver DependencyResolver) error
		}); ok {
			if resolver, invalid := newWithTag(dr, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"bindResolver: new dependency resolver at field %v (%v) failed: %w",
					field.Name, index, invalid)
			} else if invalid := b.setResolver(resolver.(DependencyResolver)); invalid != nil {
				err = fmt.Errorf(
					"bindResolver: dependency resolver %#v at field %v (%v) failed: %w",
					resolver, field.Name, index, invalid)
			}
		}
	}
	return bound, err
}

var (
	_handlerType     = TypeOf[Handler]()
	_handleCtxType   = TypeOf[HandleContext]()
	_depResolverType = TypeOf[DependencyResolver]()
	_defaultResolver = defaultDependencyResolver{}
)
