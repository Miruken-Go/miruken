package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
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

// CallbackArg returns the raw callback.
type CallbackArg struct {}

func (a CallbackArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(ctx.Callback()), nil
}

// sourceArg returns the callback source.
type sourceArg struct {}

func (a sourceArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	if src := ctx.Callback().Source(); src != nil {
		if v := reflect.ValueOf(src); v.Type().AssignableTo(typ) {
			return v, nil
		}
	}
	return reflect.ValueOf(nil), fmt.Errorf("arg: unable to resolve source: %v", typ)
}

// dependencySpec encapsulates dependency metadata.
type dependencySpec struct {
	logicalType reflect.Type
	resolver    DependencyResolver
	constraints []func(*ConstraintBuilder)
	flags       bindingFlags
	metadata    []any
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

func (s *dependencySpec) addMetadata(
	metadata any,
) error {
	s.metadata = append(s.metadata, metadata)
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

func (d DependencyArg) Promise() bool {
	return d.spec != nil && d.spec.flags & bindingPromise == bindingPromise
}

func (d DependencyArg) Metadata() []any {
	if spec := d.spec; spec != nil {
		return spec.metadata
	}
	return nil
}

func (d DependencyArg) logicalType(
	typ reflect.Type,
) reflect.Type {
	if spec := d.spec; spec != nil {
		if lt := spec.logicalType; lt != nil {
			return lt
		}
	}
	return typ
}

func (d DependencyArg) resolve(
	typ reflect.Type,
	ctx HandleContext,
) (reflect.Value, error) {
	typ = d.logicalType(typ)
	if typ == _handlerType {
		return reflect.ValueOf(ctx.Composer()), nil
	}
	if typ == _handleCtxType {
		return reflect.ValueOf(ctx), nil
	}
	callback := ctx.Callback()
	if val := reflect.ValueOf(callback); val.Type().AssignableTo(typ) {
		return val, nil
	}
	if src := callback.Source(); src != nil {
		if val := reflect.ValueOf(src); val.Type().AssignableTo(typ) {
			return val, nil
		}
	}
	var resolver DependencyResolver = &_defaultResolver
	if spec := d.spec; spec != nil {
		if spec.resolver != nil {
			resolver = spec.resolver
		}
	}
	val, err := resolver.Resolve(typ, d, ctx)
	return val, err
}

// DependencyResolver defines how an argument value is retrieved.
type DependencyResolver interface {
	Resolve(
		typ reflect.Type,
		dep DependencyArg,
		ctx HandleContext,
	) (reflect.Value, error)
}

// defaultDependencyResolver resolves the value from the Handler.
type defaultDependencyResolver struct{}

func (r *defaultDependencyResolver) Resolve(
	typ reflect.Type,
	dep DependencyArg,
	ctx HandleContext,
) (reflect.Value, error) {
	parent, _ := ctx.callback.(*Provides)
	many := !dep.Strict() && typ.Kind() == reflect.Slice
	var builder ProvidesBuilder
	builder.WithParent(parent)
	if many {
		builder.WithKey(typ.Elem())
	} else {
		builder.WithKey(typ)
	}
	if spec := dep.spec; spec != nil {
		builder.WithConstraints(spec.constraints...)
	}
	provides := builder.NewProvides()
	// TODO: async
	if result, _, err := provides.Resolve(ctx.composer, many); err == nil {
		var val reflect.Value
		if many {
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

// HandleContext contain all the information about handling a Callback.
type HandleContext struct {
	handler  any
	callback Callback
	binding  Binding
	composer Handler
	greedy   bool
}

func (h HandleContext) Handler() any {
	return h.handler
}

func (h HandleContext) Callback() Callback {
	return h.callback
}

func (h HandleContext) Binding() Binding {
	return h.binding
}

func (h HandleContext) Composer() Handler {
	return h.composer
}

func (h HandleContext) Greedy() bool {
	return h.greedy
}

// UnresolvedArgError reports a failed resolve an arg.
type UnresolvedArgError struct {
	arg    arg
	Reason error
}

func (e UnresolvedArgError) Error() string {
	return fmt.Sprintf("unresolved arg %#v: %v", e.arg, e.Reason)
}

func (e UnresolvedArgError) Unwrap() error { return e.Reason }

func resolveArgs(
	funType   reflect.Type,
	fromIndex int,
	args      []arg,
	ctx       HandleContext,
) ([]reflect.Value, *promise.Promise[[]reflect.Value], error) {
	var resolved []reflect.Value
	for i, arg := range args {
		typ := funType.In(fromIndex + i)
		if a, err := arg.resolve(typ, ctx); err != nil {
			return nil, nil, UnresolvedArgError{arg, err}
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil, nil
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
	funTyp     reflect.Type,
	startIndex int,
	endIndex   int,
	args       []arg,
	offset     int,
) (invalid error) {
	var lastSpec *dependencySpec
	for i, j := startIndex, 0; i < endIndex; i, j = i + 1, j + 1 {
		argType := funTyp.In(i)
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
				if lt, ok := promise.Inspect(argType); ok {
					if spec := arg.spec; spec == nil {
						arg.spec = &dependencySpec{flags: bindingPromise}
					} else {
						spec.flags = spec.flags | bindingPromise
					}
					arg.spec.logicalType = lt
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
			setResolver(DependencyResolver) error
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
