package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"strings"
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

// dependencyArg

type dependencyArg struct {
	spec *dependencyArgSpec
}

type dependencyArgFlags uint8

const (
	ArgStrict dependencyArgFlags = 1 << iota
	ArgOptional
)

type dependencyArgSpec struct {
	fieldIndex int
	flags      dependencyArgFlags
	resolver   DependencyArgResolver
}

type DependencyArgResolver interface {
	Resolve(
		typ          reflect.Type,
		rawCallback  interface{},
		dep         *dependencyArg,
		handler      Handler,
	) (reflect.Value, error)
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
	if v := reflect.ValueOf(rawCallback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	if spec := d.spec; spec != nil {
		if resolver := spec.resolver; resolver != nil {
			return resolver.Resolve(typ, rawCallback, d, composer)
		}
	}
	return _defaultArgResolver.Resolve(typ, rawCallback, d, composer)
}

type defaultDependencyArgResolver struct{}

func (r *defaultDependencyArgResolver) Resolve(
	typ          reflect.Type,
	rawCallback  interface{},
	dep         *dependencyArg,
	handler      Handler,
) (reflect.Value, error) {
	argType := typ
	fieldIndex := -1
	optional, strict := false, false

	if spec := dep.spec; spec != nil {
		fieldIndex = spec.fieldIndex
		optional   = spec.flags & ArgOptional == ArgOptional
		strict     = spec.flags & ArgStrict   == ArgStrict
		argType    = typ.Elem().Field(fieldIndex).Type
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
			return reflect.ValueOf(nil),
				fmt.Errorf("arg: unable to resolve dependency %v", argType)
		}
		if fieldIndex >= 0 {
			wrapper := reflect.New(typ.Elem())
			wrapper.Elem().Field(fieldIndex).Set(val)
			return wrapper, nil
		}
		return val, nil
	} else {
		return reflect.ValueOf(nil),
			fmt.Errorf("arg: unable to resolve dependency %v: %w", argType, err)
	}
}

var depArgTagParsers = []tagParser{
	tagParserFunc(parseDependencyArgOptions),
	tagParserFunc(parseDependencyArgResolver),
}

func inferDependencyArg(
	argType reflect.Type,
) (*dependencyArg, error) {
	// Is it a *Struct arg specification?
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
		if argType.Kind() == reflect.Struct && argType.Name() == "" {
			var spec *dependencyArgSpec
			if err := parseTaggedSpec(argType, &spec, depArgTagParsers); err == nil {
				if spec == nil || spec.fieldIndex < 0 {
					return &_dependencyArg, nil
				}
				dep := &dependencyArg{spec}
				if spec.resolver != nil {
					if v, ok := err.(interface {
						Validate(*dependencyArg) error
					}); ok {
						if err := v.Validate(dep); err != nil {
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

func parseDependencyArgOptions(
	index int,
	field reflect.StructField,
	spec  interface{},
) (err error) {
	if argSpec, ok := spec.(**dependencyArgSpec); ok {
		argSpecPtr := *argSpec
		if  arg, ok := field.Tag.Lookup(_argTag); ok {
			if unicode.IsLower(rune(field.Name[0])) {
				return fmt.Errorf(
					"arg: field %v at index %v must start with an uppercase character", field.Name, index)
			}
			if argSpecPtr == nil {
				argSpecPtr = new(dependencyArgSpec)
				argSpecPtr.fieldIndex = index
				*argSpec = argSpecPtr
			} else if (*argSpec).fieldIndex >= 0 {
				return fmt.Errorf(
					"arg: field %v already esignated as the argument", index)
			}
			options := strings.Split(arg, ",")
			for _, opt := range options {
				switch opt {
				case _strictOption:
					argSpecPtr.flags = argSpecPtr.flags | ArgStrict
				case _optionalOption:
					argSpecPtr.flags = argSpecPtr.flags | ArgOptional
				default:
					err = multierror.Append(err, fmt.Errorf(
						"arg: invalid option %q on field %v", opt, field.Name))
				}
			}
		}
	}
	return err
}

func parseDependencyArgResolver(
	index  int,
	field  reflect.StructField,
	spec   interface{},
) (err error) {
	if argSpec, ok := spec.(**dependencyArgSpec); ok {
		argSpecPtr := *argSpec
		if argSpecPtr == nil {
			argSpecPtr = new(dependencyArgSpec)
			argSpecPtr.fieldIndex = -1
			*argSpec = argSpecPtr
		}
		if field.Type.AssignableTo(_depArgResolverType) {
			if argSpecPtr.resolver != nil {
				argSpecPtr.resolver = reflect.New(field.Type).Interface().(DependencyArgResolver)
			} else {
				err = errors.New("arg: only one resolver is permitted")
			}
		}
	}
	return err
}

var (
	_argTag             = "arg"
	_optionalOption     = "optional"
	_zeroArg            = zeroArg{}
	_callbackArg        = callbackArg{}
	_receiverArg        = receiverArg{}
	_dependencyArg      = dependencyArg{}
	_depArgResolverType = reflect.TypeOf((*DependencyArgResolver)(nil)).Elem()
	_defaultArgResolver = defaultDependencyArgResolver{}
)
