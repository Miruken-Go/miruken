package miruken

import (
	"fmt"
	"reflect"
	"sync"
)

// Variance

type Variance uint

const(
	Covariant Variance = iota
	Contravariant
	Invariant
)

// Policy

type Policy interface {
	OrderBinding
	Variance() Variance
	AcceptResults(
		results []interface{},
	) (result interface{}, accepted HandleResult)
}

func RegisterPolicy(policy Policy) Policy {
	if policy == nil {
		panic("policy cannot be nil")
	}
	policyType := reflect.TypeOf(policy).Elem()
	if _, loaded := _policies.LoadOrStore(policyType, policy); loaded {
		panic(fmt.Sprintf("policy: %T already registered", policyType))
	}
	return policy
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

func isPolicy(typ reflect.Type) bool {
	return reflect.PtrTo(typ).Implements(_policyType)
}

func getPolicy(policyType reflect.Type) Policy {
	if policy, ok := _policies.Load(policyType); ok {
		return policy.(Policy)
	}
	return nil
}

// policySpec

type policySpec struct {
	policy     Policy
	strict     bool
	constraint interface{}
}

func (s *policySpec) setPolicy(
	policy Policy,
) error {
	s.policy = policy
	return nil
}

func (s *policySpec) setStrict(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.strict = strict
	return nil
}

var policyBuilders = []bindingBuilder{
	bindingBuilderFunc(policyBindingBuilder),
	bindingBuilderFunc(optionsBindingBuilder),
}

func buildPolicySpec(
	policyType reflect.Type,
) (spec *policySpec, err error) {
	// Is it a policy type already?
	if isPolicy(policyType) {
		if policy := getPolicy(policyType); policy != nil {
			return &policySpec{policy: policy}, nil
		}
	}
	// Is it a *Struct policy binding?
	if policyType.Kind() != reflect.Ptr {
		return spec, err
	}
	policyType = policyType.Elem()
	if policyType.Kind() == reflect.Struct &&
		policyType.Name() == "" &&  // anonymous
		policyType.NumField() > 0 {
		spec = new(policySpec)
		if err = configureBinding(policyType, spec, policyBuilders);
			err != nil || spec.policy == nil {
			return nil, err
		}
	}
	return spec, err
}

func policyBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (err error) {
	if isPolicy(field.Type) {
		if b, ok := binding.(interface {
			setPolicy(policy Policy) error
		}); ok {
			if policy := getPolicy(field.Type); policy != nil {
				if invalid := b.setPolicy(policy); invalid != nil {
					err = fmt.Errorf(
						"binding: policy %#v at index %v failed: %w",
						policy, index, invalid)
				}
			} else {
				err = fmt.Errorf(
					"binding: policy %T at index %v not found.  Did you forget to call RegisterPolicy?",
					_policyType, index)
			}
		}
	}
	return err
}

// Standard _policies

var (
	_policies       sync.Map
	_interfaceType  = reflect.TypeOf((*interface{})(nil)).Elem()
	_policyType     = reflect.TypeOf((*Policy)(nil)).Elem()
	_handleResType  = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType      = reflect.TypeOf((*error)(nil)).Elem()
	_handles        = RegisterPolicy(new(Handles))
	_provides       = RegisterPolicy(new(Provides))
	_creates        = RegisterPolicy(new(Creates))
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