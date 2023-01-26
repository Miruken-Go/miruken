package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

type (
	// Policy manages behaviors and callback Binding's.
	Policy interface {
		Filtered
		Strict() bool
		Less(binding, otherBinding Binding) bool
		IsVariantKey(key any) (bool, bool)
		MatchesKey(key, otherKey any, invariant bool) (bool, bool)
		AcceptResults(results []any) (any, HandleResult)
	}

	// PolicyDispatch customizes Callback Policy dispatch.
	PolicyDispatch interface {
		DispatchPolicy(
			policy   Policy,
			callback Callback,
			greedy   bool,
			composer Handler,
		) HandleResult
	}

	// policyKey binds a Policy to a key for lookup.
	policyKey struct {
		policy Policy
		key    any
	}
)

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
	policies    []policyKey
	flags       bindingFlags
	filters     []FilterProvider
	constraints []BindingConstraint
	metadata    []any
	arg         arg
}

func (s *policySpec) addPolicy(
	policy Policy,
	field  reflect.StructField,
) error {
	pk := policyKey{policy: policy}
	if key, ok := field.Tag.Lookup("key"); ok {
		for _, pk := range s.policies {
			if pk.policy == policy && pk.key == key {
				return fmt.Errorf("duplicate key \"%s\"", key)
			}
		}
		pk.key = key
	}
	s.policies = append(s.policies, pk)
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

func (s *policySpec) addMetadata(
	metadata any,
) error {
	s.metadata = append(s.metadata, metadata)
	return nil
}

func (s *policySpec) complete() error {
	provider := ConstraintProvider{s.constraints}
	s.filters = append(s.filters, &provider)
	return nil
}

// policySpecBuilder builds policySpec from method metadata.
type policySpecBuilder struct {
	cache   map[reflect.Type]Policy
	parsers []bindingParser
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
			if err = parseBinding(specType, spec, p.parsers);
				err != nil || len(spec.policies) == 0 {
				return nil, err
			}
			spec.arg = zeroArg{} // spec is just a placeholder
			return spec, nil
		}
	}
	// Is it a Callback arg?
	if callbackOrSpec.Kind() != reflect.Interface && callbackOrSpec.Implements(callbackType) {
		return &policySpec{
			policies: []policyKey{{policy: p.policyOf(callbackOrSpec)}},
			arg:      CallbackArg{},
			filters:  []FilterProvider{&ConstraintProvider{}},
		}, nil
	}
	return nil, nil
}

func (p *policySpecBuilder) policyOf(
	callbackType reflect.Type,
) Policy {
	if p.cache == nil {
		p.cache = make(map[reflect.Type]Policy)
	} else if policy, ok := p.cache[callbackType]; ok {
		return policy
	}
	policy := reflect.Zero(callbackType).Interface().(Callback).Policy()
	p.cache[callbackType] = policy
	return policy
}

func (p *policySpecBuilder) parse(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if cb := coerceToPtr(typ, callbackType); cb != nil {
		if typ.Kind() == reflect.Interface {
			return false, fmt.Errorf("callback cannot be an interface: %v", typ)
		}
		bound = true
		if b, ok := binding.(interface {
			addPolicy(Policy, reflect.StructField) error
		}); ok {
			policy := p.policyOf(cb)
			if invalid := b.addPolicy(policy, field); invalid != nil {
				err = fmt.Errorf(
					"parse: %v at index %v failed: %w", typ, index, invalid)
			}
		}
	}
	return bound, err
}

func parseFilters(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if filter := coerceToPtr(typ, filterType); filter != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(FilterProvider) error
		}); ok {
			spec := filterSpec{filter, false, -1}
			if f, ok := field.Tag.Lookup(filterTag); ok {
				args := strings.Split(f, ",")
				for _, arg := range args {
					if arg == requiredArg {
						spec.required = true
					} else {
						if count, _ := fmt.Sscanf(arg, "order=%d", &spec.order); count > 0 {
							continue
						}
					}
				}
			}
			provider := &filterSpecProvider{spec}
			if invalid := b.addFilterProvider(provider); invalid != nil {
				err = fmt.Errorf(
					"parseFilters: filter spec provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	} else if fp := coerceToPtr(typ, filterProviderType); fp != nil {
		bound = true
		if b, ok := binding.(interface {
			addFilterProvider(FilterProvider) error
		}); ok {
			if provider, invalid := newWithTag(fp, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"parseFilters: new filter provider at index %v failed: %w",
					index, invalid)
			} else if invalid := b.addFilterProvider(provider.(FilterProvider)); invalid != nil {
				err = fmt.Errorf(
					"parseFilters: filter provider %v at index %v failed: %w",
					provider, index, invalid)
			}
		}
	}
	return bound, err
}

func parseConstraints(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if ct := coerceToPtr(typ, constraintType); ct != nil {
		bound = true
		if b, ok := binding.(interface {
			addConstraint(BindingConstraint) error
		}); ok {
			if constraint, invalid := newWithTag(ct, field.Tag); invalid != nil {
				err = fmt.Errorf(
					"parseConstraints: new key at index %v failed: %w", index, invalid)
			} else if invalid := b.addConstraint(constraint.(BindingConstraint)); invalid != nil {
				err = fmt.Errorf(
					"parseConstraints: key %v at index %v failed: %w", constraint, index, invalid)
			}
		}
	}
	return bound, err
}

const (
	filterTag   = "filter"
	requiredArg = "required"
)

var (
	anyType             = TypeOf[any]()
	anySliceType        = TypeOf[[]any]()
	promiseAnySliceType = TypeOf[*promise.Promise[[]any]]()
	errorType           = TypeOf[error]()
	callbackType        = TypeOf[Callback]()
	filterType          = TypeOf[Filter]()
	filterProviderType  = TypeOf[FilterProvider]()
	constraintType      = TypeOf[BindingConstraint]()
	handleResType       = TypeOf[HandleResult]()
)
