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
	IsVariantKey(key interface{}) (bool, bool)
	MatchesKey(key, otherKey interface{}, strict bool) (bool, bool)
	AcceptResults(results []interface{}) (interface{}, HandleResult)
}

// PolicyDispatch allows handlers to override callback dispatch.
type PolicyDispatch interface {
	DispatchPolicy(
		policy      Policy,
		callback    interface{},
		rawCallback Callback,
		greedy      bool,
		composer    Handler,
	) HandleResult
}

func DispatchPolicy(
	handler     interface{},
	callback    interface{},
	rawCallback Callback,
	greedy      bool,
	composer    Handler,
) HandleResult {
	policy := rawCallback.Policy()
	if dp, ok := handler.(PolicyDispatch); ok {
		return dp.DispatchPolicy(
			policy, callback, rawCallback, greedy, composer)
	}
	if factory := GetHandlerDescriptorFactory(composer); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if d, err := factory.HandlerDescriptorOf(handlerType); d != nil {
			return d.Dispatch(policy, handler, callback, rawCallback, greedy, composer)
		} else if err != nil {
			return NotHandled.WithError(err)
		}
	}
	return NotHandled
}

// policySpec captures Policy metadata.
type policySpec struct {
	policies []Policy
	flags    bindingFlags
	filters  []FilterProvider
	key      interface{}
	arg      arg
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

// policySpecBuilder builds policySpec from method metadata.
type policySpecBuilder struct {
	cache map[reflect.Type]Policy
}

func (p *policySpecBuilder) BuildSpec(
	callbackOrSpec reflect.Type,
) (spec *policySpec, err error) {
	// Is it a policy spec?
	if callbackOrSpec.Kind() == reflect.Ptr {
		if specType := callbackOrSpec.Elem();
			specType.Name() == "" && // anonymous
			specType.Kind() == reflect.Struct &&
			specType.NumField() > 0 {
			spec = &policySpec{}
			builders := []bindingBuilder{
				bindingBuilderFunc(p.bindPolicies),
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
		return &policySpec{
			policies: []Policy{p.policyOf(callbackOrSpec)},
			arg:      rawCallbackArg{},
		}, nil
	}
	return nil, nil
}

func (p *policySpecBuilder) bindPolicies(
	index   int,
	field   reflect.StructField,
	binding interface{},
) (bound bool, err error) {
	if cb := coerceToPtr(field.Type, _callbackType); cb != nil {
		bound = true
		if b, ok := binding.(interface {
			addPolicy(policy Policy) error
		}); ok {
			policy := p.policyOf(cb)
			if invalid := b.addPolicy(policy); invalid != nil {
				err = fmt.Errorf(
					"bindPolicies: policy %#v at index %v failed: %w",
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
					"bindFilters: filter spec provider %v at index %v failed: %w",
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
	_interfaceType      = reflect.TypeOf((*interface{})(nil)).Elem()
	_interfaceSliceType = reflect.TypeOf((*[]interface{})(nil)).Elem()
	_callbackType       = reflect.TypeOf((*Callback)(nil)).Elem()
	_filterType         = reflect.TypeOf((*Filter)(nil)).Elem()
	_filterProviderType = reflect.TypeOf((*FilterProvider)(nil)).Elem()
	_constraintType     = reflect.TypeOf((*BindingConstraint)(nil)).Elem()
	_handleResType      = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
)
