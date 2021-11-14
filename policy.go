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
	Invariant Variance = 0
	Covariant Variance = 1 << iota
	Contravariant
	Bivariant = Covariant | Contravariant
)

// Policy defines behaviors for callbacks.
type Policy interface {
	OrderBinding
	Filtered
	Variance() Variance
	AcceptResults(results []interface{}) (interface{}, HandleResult)
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
		rawCallback Callback,
		greedy      bool,
		composer    Handler,
		results     ResultReceiver,
	) HandleResult
}

func DispatchPolicy(
	policy      Policy,
	handler     interface{},
	callback    interface{},
	rawCallback Callback,
	greedy      bool,
	composer    Handler,
	results     ResultReceiver,
) HandleResult {
	if dp, ok := handler.(PolicyDispatch); ok {
		return dp.DispatchPolicy(
			policy, callback, rawCallback, greedy, composer, results)
	}
	if factory := GetHandlerDescriptorFactory(composer); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if d, err := factory.GetHandlerDescriptor(handlerType); d != nil {
			context := HandleContext{callback, rawCallback, composer, results}
			return d.Dispatch(policy, handler, greedy, context)
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

// policySpec encapsulates policy metadata.
type policySpec struct {
	policies    []Policy
	flags       bindingFlags
	filters     []FilterProvider
	key         interface{}
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

func (s *policySpec) addConstraint(
	constraint BindingConstraint,
) error {
	provider := ConstraintProvider{constraint}
	s.filters = append(s.filters, &provider)
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

func (s *policySpec) unknownBinding(
	index int,
	field reflect.StructField,
) error {
	return nil
}

var policyBuilders = []bindingBuilder{
	bindingBuilderFunc(policyBindingBuilder),
	bindingBuilderFunc(optionsBindingBuilder),
	bindingBuilderFunc(filterBindingBuilder),
	bindingBuilderFunc(constraintBindingBuilder),
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
	// Is it a *struct policy binding?
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
) (bound bool, err error) {
	if isPolicy(field.Type) {
		bound = true
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
	return bound, err
}

func filterBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	typ := field.Type
	if filter := coerceToPtr(typ, _filterType); filter != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(provider FilterProvider) error
		}); ok {
			spec := filterSpec{filter, false, -1}
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
			provider := &FilterSpecProvider{spec}
			if invalid := b.addFilterProvider(provider); invalid != nil {
				err = fmt.Errorf(
					"binding: filter spec provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	} else if fp := coerceToPtr(typ, _filterProviderType); fp != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(provider FilterProvider) error
		}); ok {
			if provider, invalid := newWithTag(fp, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"binding: new filter provider at index %v failed: %w",
					index, invalid)
			} else if invalid := b.addFilterProvider(provider.(FilterProvider)); invalid != nil {
				err = fmt.Errorf(
					"binding: filter provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	}
	return bound, err
}

func constraintBindingBuilder(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	typ := field.Type
	if ct := coerceToPtr(typ, _constraintType); ct != nil {
		bound = true
		if b, ok := binding.(interface {
			addConstraint(BindingConstraint) error
		}); ok {
			if constraint, invalid := newWithTag(ct, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"binding: new key at index %v failed: %w",
					index, invalid)
			} else if invalid := b.addConstraint(constraint.(BindingConstraint)); invalid != nil {
				err = fmt.Errorf(
					"binding: key %v at index %v failed: %w",
					constraint, index, invalid)
			}
		}
	}
	return bound, err
}

var (
	_policies           sync.Map
	_filterTag          = "filter"
	_requiredArg        = "required"
	_interfaceType      = reflect.TypeOf((*interface{})(nil)).Elem()
	_interfaceSliceType = reflect.TypeOf((*[]interface{})(nil)).Elem()
	_policyType         = reflect.TypeOf((*Policy)(nil)).Elem()
	_filterType         = reflect.TypeOf((*Filter)(nil)).Elem()
	_filterProviderType = reflect.TypeOf((*FilterProvider)(nil)).Elem()
	_constraintType     = reflect.TypeOf((*BindingConstraint)(nil)).Elem()
	_handleResType      = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
)
