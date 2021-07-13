package miruken

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Variance determines how callbacks are handled.
type Variance uint

const (
	Covariant Variance = iota
	Contravariant
	Invariant
)

// Policy defines behaviors for callbacks.
type Policy interface {
	OrderBinding
	Filtered
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

// PolicyDispatch allows handlers to override callback dispatch.
type PolicyDispatch interface {
	DispatchPolicy(
		policy      Policy,
		callback    interface{},
		rawCallback interface{},
		constraint  interface{},
		greedy      bool,
		composer    Handler,
		results     ResultReceiver,
	) HandleResult
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
	if dp, ok := handler.(PolicyDispatch); ok {
		return dp.DispatchPolicy(
			policy, callback, rawCallback,
			constraint, greedy, composer, results)
	}
	if factory := GetHandlerDescriptorFactory(composer); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if d, err := factory.GetHandlerDescriptor(handlerType); d != nil {
			if rawCallback == nil {
				rawCallback = callback
			}
			context := HandleContext{callback, rawCallback, composer, results}
			return d.Dispatch(policy, handler, constraint, greedy, context)
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

// policySpec represents policy metadata
type policySpec struct {
	policies    []Policy
	flags       bindingFlags
	filters     []FilterProvider
	constraint  interface{}
}

func (s *policySpec) addPolicy(
	policy Policy,
) error {
	s.policies = append(s.policies, policy)
	return nil
}

func (s *policySpec) addFilterProvider(
	provider FilterProvider,
) error {
	s.filters = append(s.filters, provider)
	return nil
}

func (s *policySpec) setStrict(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.flags = s.flags | bindingStrict
	return nil
}

func (s *policySpec) setSkipFilters(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	s.flags = s.flags | bindingSkipFilters
	return nil
}

var policyBuilders = []bindingBuilder{
	bindingBuilderFunc(policyBindingBuilder),
	bindingBuilderFunc(optionsBindingBuilder),
	bindingBuilderFunc(filterBindingBuilder),
}

func buildPolicySpec(
	policyType reflect.Type,
) (spec *policySpec, err error) {
	// Is it a policy type already?
	if isPolicy(policyType) {
		if policy := getPolicy(policyType); policy != nil {
			return &policySpec{policies: []Policy{policy}}, nil
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
			err != nil || len(spec.policies) == 0 {
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
			addPolicy(policy Policy) error
		}); ok {
			if policy := getPolicy(field.Type); policy != nil {
				if invalid := b.addPolicy(policy); invalid != nil {
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

func filterBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (err error) {
	if field.Type.Implements(_filterType) {
		if p, ok := binding.(interface {
			addFilterProvider(provider FilterProvider) error
		}); ok {
			spec := filterSpec{field.Type, false, -1}
			if f, ok := field.Tag.Lookup(_filterTag); ok {
				args := strings.Split(f, ",")
				for _, arg := range args {
					if arg == _requiredArg {
						spec.required = true
					} else {
						if count, _ := fmt.Sscanf(arg, "order=%d", &spec.order); count > 0 {
							continue
						}
					}
				}
			}
			provider := &filterSpecProvider{spec}
			if invalid := p.addFilterProvider(provider); invalid != nil {
				err = fmt.Errorf(
					"binding: filter spec provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	} else if field.Type.Implements(_filterProviderType) {
		if p, ok := binding.(interface {
			addFilterProvider(provider FilterProvider) error
		}); ok {
			provider := newStructField(field).(FilterProvider)
			if invalid := p.addFilterProvider(provider); invalid != nil {
				err = fmt.Errorf(
					"binding: filter provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	}
	return err
}

// Standard _policies

var (
	_policies           sync.Map
	_filterTag          = "filter"
	_requiredArg        = "required"
	_interfaceType      = reflect.TypeOf((*interface{})(nil)).Elem()
	_policyType         = reflect.TypeOf((*Policy)(nil)).Elem()
	_filterType         = reflect.TypeOf((*Filter)(nil)).Elem()
	_filterProviderType = reflect.TypeOf((*FilterProvider)(nil)).Elem()
	_handleResType      = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
	_handles            = RegisterPolicy(new(Handles))
	_provides           = RegisterPolicy(new(Provides))
	_creates            = RegisterPolicy(new(Creates))
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
func (p *Provides) newConstructorBinding(
	handlerType  reflect.Type,
	initMethod  *reflect.Method,
	spec        *policySpec,
) (binding Binding, invalid error) {
	return newConstructorBinding(handlerType, initMethod, spec)
}
func ProvidesPolicy() Policy { return _provides }

// Creates policy for creating instances covariantly.
type Creates struct {
	covariantPolicy
}
func (p *Creates) newConstructorBinding(
	handlerType  reflect.Type,
	initMethod  *reflect.Method,
	spec        *policySpec,
) (binding Binding, invalid error) {
	return newConstructorBinding(handlerType, initMethod, spec)
}
func CreatesPolicy() Policy { return _creates }