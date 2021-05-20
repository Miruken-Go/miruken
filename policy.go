package miruken

import (
	"fmt"
	"github.com/fatih/structtag"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"strconv"
	"sync"
)

// Policy

type Policy interface {
	OrderBinding
	Variance() Variance

	AcceptResults(
		results []interface{},
	) (result interface{}, accepted HandleResult)
}

func DispatchPolicy(
	policy      Policy,
	handler     interface{},
	callback    interface{},
	rawCallback interface{},
	constraint  interface{},
	greedy      bool,
	composer    Handler,
	results     ResultReceiver,
) HandleResult {
	// The HandlerDescriptorFactory is resolved from a Handler
	// so when requesting it, we don't want to resolve itself
	if _, ok := callback.(*getHandlerDescriptorFactory); ok {
		return NotHandled
	}
	if factory := GetHandlerDescriptorFactory(composer); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if d, err := factory.GetHandlerDescriptor(handlerType); d != nil {
			if rawCallback == nil {
				rawCallback = callback
			}
			return d.Dispatch(
				policy, handler, callback, rawCallback,
				constraint, greedy, composer, results)
		} else if err != nil {
			return NotHandled.WithError(err)
		}
	}
	return NotHandled
}

func RegisterPolicy(policy Policy) Policy {
	if policy == nil {
		panic("policy cannot be nil")
	}
	policyType := reflect.TypeOf(policy).Elem()
	if _, loaded := _policies.LoadOrStore(policyType, policy); loaded {
		panic(fmt.Sprintf("policy: %v already registered", policyType))
	}
	return policy
}

func isPolicy(typ reflect.Type) bool {
	return reflect.PtrTo(typ).Implements(_policyType)
}

func getPolicy(policyType reflect.Type) Policy {
	if policy, ok := _policies.Load(policyType); ok {
		return policy.(Policy)
	}
	return nil
}

func inferBinding(
	bindingType reflect.Type,
) (policy Policy, spec *bindingSpec, err error) {
	var policyType reflect.Type
	// Is it a policy type already?
	if isPolicy(bindingType) {
		policyType = bindingType
		if policy = getPolicy(policyType); policy != nil {
			return policy, new(bindingSpec), nil
		}
	}
	// Is it a binding specification?
	if bindingType.Kind() == reflect.Ptr {
		bindingType = bindingType.Elem()
	}
	if bindingType.Kind() == reflect.Struct && bindingType.NumField() > 0 {
		field := bindingType.Field(0)
		if isPolicy(field.Type) {
			policyType = field.Type
			if policy = getPolicy(policyType); policy != nil {
				spec, err := parseBindingSpec(bindingType)
				return policy, spec, err
			}
		}
	}
	if policyType != nil {
		panic(fmt.Sprintf("policy: %v not found.  Did you forget to call RegisterPolicy?", policyType))
	}
	return nil, nil, nil
}

type parseTagFunc func (
	index  int,
	field  reflect.StructField,
	tags  *structtag.Tags,
	spec  *bindingSpec,
) error

var tagParsers = []parseTagFunc{parseStrictTag}

func parseBindingSpec(
	specType reflect.Type,
) (spec *bindingSpec, err error) {
	spec = new(bindingSpec)
	for i := 0; i < specType.NumField(); i++ {
		for _, parser := range tagParsers {
			field := specType.Field(i)
			tags, invalid := structtag.Parse(string(field.Tag))
			if invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"binding: invalid tag %v on field %v %w",
					field.Tag, field.Name, invalid))
			}
			if invalid := parser(i, field, tags, spec); invalid != nil {
				err = multierror.Append(err, invalid)
			}
		}
	}
	return spec, err
}

func parseStrictTag(
	index  int,
	field  reflect.StructField,
	tags  *structtag.Tags,
	spec  *bindingSpec,
) error {
	if index != 0 {
		return nil
	}
	if strictTag, _ := tags.Get(_strictTag); strictTag != nil {
		if strict, err := strconv.ParseBool(strictTag.Name); err == nil {
			spec.strict = strict
		} else {
			return fmt.Errorf("binding: invalid tag %q on field %v %w",
				strictTag, field.Name, err)
		}
	}
	return nil
}

// Standard _policies

var (
	_policies sync.Map
	_strictTag     = "strict"
	_interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
	_policyType    = reflect.TypeOf((*Policy)(nil)).Elem()
	_handleResType = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType     = reflect.TypeOf((*error)(nil)).Elem()
	_handles       = RegisterPolicy(new(Handles))
	_provides      = RegisterPolicy(new(Provides))
	_creates       = RegisterPolicy(new(Creates))
)

// Handles policy for handling callbacks contravariantly.
type Handles struct {
	contravariantPolicy
}
func HandlesPolicy() Policy { return _handles }

// Provides policy for providing instances covariantly.
type Provides struct {
	covariantPolicy
}
func ProvidesPolicy() Policy { return _provides }

// Creates policy for creating instances covariantly.
type Creates struct {
	covariantPolicy
}
func CreatesPolicy() Policy { return _creates }