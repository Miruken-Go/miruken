package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

type (
	// Binding connects a Callback to a handler.
	Binding interface {
		Filtered
		Key()               any
		Strict()            bool
		SkipFilters()       bool
		Async()             bool
		Exported()          bool
		Metadata()          []any
		LogicalOutputType() reflect.Type
		Invoke(
			ctx HandleContext,
			initArgs ...any,
		) ([]any, *promise.Promise[[]any], error)
	}

	// BindingBase implements common binding contract.
	BindingBase struct {
		FilteredScope
		flags    bindingFlags
		metadata []any
	}

	// BindingReducer aggregates Binding results.
	BindingReducer func(
		binding Binding,
		result  HandleResult,
	) (HandleResult, bool)

	// Late is a container for late Binding results.
	Late struct {
		Value any
	}

	BindingParser interface {
		parse(
			index   int,
			field   reflect.StructField,
			binding any,
		) (bound bool, err error)
	}

	BindingParserFunc func (
		index   int,
		field   reflect.StructField,
		binding any,
	) (bound bool, err error)

	// Strict Binding's do not expand results.
	Strict struct{}

	// Optional marks a dependency not required.
	Optional struct{}

	// SkipFilters skips all non-required filters.
	SkipFilters struct{}

	// BindingGroup marks bindings that aggregate
	// one or more binding metadata.
	BindingGroup struct {}
)


// BindingGroup

func (BindingGroup) DefinesBindingGroup() {}


// BindingBase

func (b *BindingBase) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *BindingBase) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *BindingBase) Async() bool {
	return b.flags & bindingAsync == bindingAsync
}

func (b *BindingBase) Metadata() []any {
	return b.metadata
}


// BindingParserFunc

func (b BindingParserFunc) parse(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	return b(index, field, binding)
}

type (
	// bindingSpec captures a Binding specification.
	bindingFlags uint8

	bindingSpec struct {
		policies    []policyKey
		flags       bindingFlags
		filters     []FilterProvider
		constraints []BindingConstraint
		metadata    []any
		arg         arg
		lt          reflect.Type
	}

	// bindingSpecFactory creates bindingSpec's from type metadata.
	bindingSpecFactory struct {
		cache   map[reflect.Type]Policy
		parsers []BindingParser
	}
)

const (
	bindingNone bindingFlags = 0
	bindingStrict bindingFlags = 1 << iota
	bindingOptional
	bindingSkipFilters
	bindingAsync
)


// bindingSpec

func (b *bindingSpec) addPolicy(
	policy Policy,
	field  reflect.StructField,
) error {
	pk := policyKey{policy: policy}
	if key, ok := field.Tag.Lookup("key"); ok {
		for _, pk := range b.policies {
			if pk.policy == policy && pk.key == key {
				return fmt.Errorf("duplicate key \"%s\"", key)
			}
		}
		pk.key = key
	}
	b.policies = append(b.policies, pk)
	return nil
}

func (b *bindingSpec) addFilterProvider(
	provider FilterProvider,
) error {
	b.filters = append(b.filters, provider)
	return nil
}

func (b *bindingSpec) addConstraint(
	constraint BindingConstraint,
) error {
	for _, c := range b.constraints {
		if merge, ok := c.(interface {
			Merge(BindingConstraint) bool
		}); ok && merge.Merge(constraint) {
			return nil
		}
	}
	b.constraints = append(b.constraints, constraint)
	return nil
}

func (b *bindingSpec) setStrict(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	b.flags = b.flags | bindingStrict
	return nil
}

func (b *bindingSpec) setSkipFilters(
	index  int,
	field  reflect.StructField,
	strict bool,
) error {
	b.flags = b.flags | bindingSkipFilters
	return nil
}

func (b *bindingSpec) addMetadata(
	metadata any,
) error {
	b.metadata = append(b.metadata, metadata)
	return nil
}

func (b *bindingSpec) setLogicalOutputType(lt reflect.Type) {
	switch lt {
	case errorType, handleResType: break
	default:
		b.lt = lt
	}
}

func (b *bindingSpec) complete() error {
	provider := ConstraintProvider{b.constraints}
	b.filters = append(b.filters, &provider)
	return nil
}


// bindingSpecFactory

func (p *bindingSpecFactory) createSpec(
	callbackOrSpec reflect.Type,
) (spec *bindingSpec, err error) {
	// Is it a policy spec?
	if callbackOrSpec.Kind() == reflect.Ptr {
		if specType := callbackOrSpec.Elem();
			specType.Name() == "" && // anonymous
				specType.Kind() == reflect.Struct &&
				specType.NumField() > 0 {
			spec = &bindingSpec{}
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
		return &bindingSpec{
			policies: []policyKey{{policy: p.policyOf(callbackOrSpec)}},
			arg:      CallbackArg{},
			filters:  []FilterProvider{&ConstraintProvider{}},
		}, nil
	}
	return nil, nil
}

func (p *bindingSpecFactory) policyOf(
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

func (p *bindingSpecFactory) parse(
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

func parseBinding(
	source  reflect.Type,
	binding any,
	parsers []BindingParser,
) (err error) {
	if err = parseStructBinding(source, binding, parsers); err == nil {
		if b, ok := binding.(interface {
			complete() error
		}); ok {
			err = b.complete()
		}
	}
	return
}

func parseStructBinding(
	typ     reflect.Type,
	binding any,
	parsers []BindingParser,
) (err error) {
	checkedMetadata := false
	var metadataOwner interface {
		addMetadata(metadata any) error
	}
	NextField:
	for i := 0; i < typ.NumField(); i++ {
		bound := false
		field := typ.Field(i)
		fieldType := field.Type
		if fieldType == bindingGroupType {
			continue
		}
		if fieldType.Kind() == reflect.Struct && fieldType.Implements(definesBindingGroup) {
			if invalid := parseStructBinding(fieldType, binding, parsers); invalid != nil {
				err = multierror.Append(err, invalid)
			}
			continue
		}
		for _, parser := range parsers {
			if b, invalid := parser.parse(i, field, binding); invalid != nil {
				err = multierror.Append(err, invalid)
				continue NextField
			} else if b {
				bound = true
				break
			}
		}
		if !bound && (metadataOwner != nil || !checkedMetadata) {
			if !checkedMetadata {
				checkedMetadata = true
				metadataOwner, _ = binding.(interface {
					addMetadata(metadata any) error
				})
			}
			if metadataOwner != nil {
				if invalid := addMetadata(field.Type, field.Tag, metadataOwner); invalid != nil {
					err = multierror.Append(err, invalid)
				}
			}
		}
	}
	return err
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

func parseOptions(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if typ == strictType {
		bound = true
		if b, ok := binding.(interface {
			setStrict(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setStrict(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: strict field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == optionalType {
		bound = true
		if b, ok := binding.(interface {
			setOptional(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setOptional(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: optional field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == skipFiltersType {
		bound = true
		if b, ok := binding.(interface {
			setSkipFilters(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setSkipFilters(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: skipFilters on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	}
	return bound, err
}

func addMetadata(
	typ   reflect.Type,
	tag   reflect.StructTag,
	owner interface {
		addMetadata(metadata any) error
	},
) error {
	writeable := typ.Kind() == reflect.Ptr
	if !writeable {
		typ = reflect.PtrTo(typ)
	}
	if metadata, err := newWithTag(typ, tag); metadata != nil && err == nil {
		if !writeable {
			metadata = reflect.Indirect(reflect.ValueOf(metadata)).Interface()
		}
		return owner.addMetadata(metadata)
	} else {
		return err
	}
}


const (
	filterTag   = "filter"
	requiredArg = "required"
)

var (
	strictType          = TypeOf[Strict]()
	optionalType        = TypeOf[Optional]()
	skipFiltersType     = TypeOf[SkipFilters]()
	bindingGroupType    = TypeOf[BindingGroup]()
	definesBindingGroup = TypeOf[interface{ DefinesBindingGroup() }]()
	filterType          = TypeOf[Filter]()
	constraintType      = TypeOf[BindingConstraint]()
)