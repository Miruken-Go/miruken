package miruken

import (
	"fmt"
	"reflect"
	"strings"
)

// Policy manages behaviors and callback Binding's.
type Policy interface {
	Filtered
	Less(binding, otherBinding Binding) bool
	IsVariantKey(key any) (bool, bool)
	MatchesKey(key, otherKey any, strict bool) (bool, bool)
	AcceptResults(results []any) (any, HandleResult)
}

// PolicyDispatch customizes Callback Policy dispatch.
type PolicyDispatch interface {
	DispatchPolicy(
		policy   Policy,
		callback Callback,
		greedy   bool,
		composer Handler,
	) HandleResult
}

func DispatchPolicy(
	handler  any,
	callback Callback,
	greedy   bool,
	composer Handler,
) HandleResult {
	policy := callback.Policy()
	if dp, ok := handler.(PolicyDispatch); ok {
		return dp.DispatchPolicy(policy, callback, greedy, composer)
	}
	if factory := GetHandlerDescriptorFactory(composer); factory != nil {
		if d := factory.DescriptorOf(handler); d != nil {
			return d.Dispatch(policy, handler, callback, greedy, composer, nil)
		}
	}
	return NotHandled
}

// policySpec captures Policy metadata.
type policySpec struct {
	policies    []Policy
	flags       bindingFlags
	filters     []FilterProvider
	constraints []BindingConstraint
	key         any
	arg         arg
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
	for _, c := range s.constraints {
		if merge, ok := c.(interface {
			Merge(BindingConstraint) bool
		}); ok && merge.Merge(constraint) {
			return nil
		}
	}
	s.constraints = append(s.constraints, constraint)
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

func (s *policySpec) complete() error {
	if len(s.constraints) > 0 {
		provider := ConstraintProvider{s.constraints}
		s.filters = append(s.filters, &provider)
	}
	return nil
}

// policySpecBuilder builds policySpec from method metadata.
type policySpecBuilder struct {
	cache    map[reflect.Type]Policy
	builders []bindingBuilder
}

func (p *policySpecBuilder) buildSpec(
	callbackOrSpec reflect.Type,
) (spec *policySpec, err error) {
	// Is it a policy spec?
	if callbackOrSpec.Kind() == reflect.Ptr {
		if specType := callbackOrSpec.Elem();
			specType.Name() == "" && // anonymous
			specType.Kind() == reflect.Struct &&
			specType.NumField() > 0 {
			spec = &policySpec{}
			builders := []bindingBuilder{p,
				bindingBuilderFunc(bindOptions),
				bindingBuilderFunc(bindFilters),
				bindingBuilderFunc(bindConstraints),
			}
			if err = configureBinding(specType, spec, builders);
				err != nil || len(spec.policies) == 0 {
				return nil, err
			}
			spec.arg = zeroArg{} // spec is just a placeholder
			return spec, nil
		}
	}
	// Is it a callback arg?
	if callbackOrSpec.Implements(_callbackType) {
		if callbackOrSpec.Kind() == reflect.Interface {
			return nil, fmt.Errorf("callback argument cannot be an interface: %v", callbackOrSpec)
		} else {
			return &policySpec{
				policies: []Policy{p.policyOf(callbackOrSpec)},
				arg:      CallbackArg{},
			}, nil
		}
	}
	return nil, nil
}

func (p *policySpecBuilder) configure(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if cb := coerceToPtr(typ, _callbackType); cb != nil {
		if typ.Kind() == reflect.Interface {
			return false, fmt.Errorf("callback cannot be an interface: %v", typ)
		}
		bound = true
		if b, ok := binding.(interface {
			addPolicy(Policy) error
		}); ok {
			policy := p.policyOf(cb)
			if invalid := b.addPolicy(policy); invalid != nil {
				err = fmt.Errorf(
					"configure: policy %#v at index %v failed: %w",
					policy, index, invalid)
			}
		}
	}
	return bound, err
}

func (p *policySpecBuilder) policyOf(
	callbackType reflect.Type,
) Policy {
	if p.cache == nil {
		p.cache = make(map[reflect.Type]Policy)
	} else if policy, ok := p.cache[callbackType]; ok {
		return policy
	}
	if pm, ok := callbackType.MethodByName("Policy"); ok {
		val := pm.Func.Call([]reflect.Value{reflect.Zero(callbackType)})
		policy := val[0].Interface().(Policy)
		p.cache[callbackType] = policy
		return policy
	}
	panic(fmt.Sprintf("missing Policy() method for callback %v ", callbackType))
}

func bindFilters(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if filter := coerceToPtr(typ, _filterType); filter != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(FilterProvider) error
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
					"bindFilters: filter spec provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	} else if fp := coerceToPtr(typ, _filterProviderType); fp != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(FilterProvider) error
		}); ok {
			if provider, invalid := newWithTag(fp, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"bindFilters: new filter provider at index %v failed: %w",
					index, invalid)
			} else if invalid := b.addFilterProvider(provider.(FilterProvider)); invalid != nil {
				err = fmt.Errorf(
					"bindFilters: filter provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	}
	return bound, err
}

func bindConstraints(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if ct := coerceToPtr(typ, _constraintType); ct != nil {
		bound = true
		if b, ok := binding.(interface {
			addConstraint(BindingConstraint) error
		}); ok {
			if constraint, invalid := newWithTag(ct, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"bindConstraints: new key at index %v failed: %w", index, invalid)
			} else if invalid := b.addConstraint(constraint.(BindingConstraint)); invalid != nil {
				err = fmt.Errorf(
					"bindConstraints: key %v at index %v failed: %w", constraint, index, invalid)
			}
		}
	}
	return bound, err
}

var (
	_filterTag          = "filter"
	_requiredArg        = "required"
	_anyType            = TypeOf[any]()
	_anySliceType       = TypeOf[[]any]()
	_errorType          = TypeOf[error]()
	_callbackType       = TypeOf[Callback]()
	_filterType         = TypeOf[Filter]()
	_filterProviderType = TypeOf[FilterProvider]()
	_constraintType     = TypeOf[BindingConstraint]()
	_handleResType      = TypeOf[HandleResult]()
)
