package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

type (
	// KeyValues is an alias for key/value map.
	KeyValues map[any]any

	// BindingMetadata is a key/value container.
	BindingMetadata struct {
		name   string
		values KeyValues
	}
)

func (b *BindingMetadata) Name() string {
	return b.name
}

func (b *BindingMetadata) SetName(name string) {
	b.name = name
}

func (b *BindingMetadata) IsEmpty() bool {
	return b.name == "" && len(b.values) == 0
}

func (b *BindingMetadata) Has(key any) bool {
	if key == nil {
		panic("key cannot be nil")
	}
	if b.values == nil {
		return false
	}
	_, ok := b.values[key]
	return ok
}

func (b *BindingMetadata) Get(key any) (any, bool) {
	if key == nil {
		panic("key cannot be nil")
	}
	if b.values == nil {
		return nil, false
	}
	val, ok := b.values[key]
	return val, ok
}

func (b *BindingMetadata) Set(key any, val any) {
	if key == nil {
		panic("key cannot be nil")
	}
	if b.values == nil {
		b.values = make(KeyValues)
	}
	b.values[key] = val
}

func (b* BindingMetadata) MergeInto(other *BindingMetadata) {
	if other == nil {
		panic("other cannot be nil")
	}
	if b.values != nil {
		for key, val := range b.values {
			other.Set(key, val)
		}
	}
}

// BindingScope is a BindingMetadata container.
type BindingScope interface {
	Metadata() *BindingMetadata
}

// BindingConstraint manages BindingMetadata assertions.
type BindingConstraint interface {
	Require(metadata *BindingMetadata)
	Matches(metadata *BindingMetadata) bool
}

// Named matches against a name.
type Named string

func (n *Named) Name() string {
	return string(*n)
}

func (n *Named) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok && len(strings.TrimSpace(name)) > 0 {
		*n = Named(name)
		return nil
	}
	return ErrNameMissing
}

func (n *Named) Require(metadata *BindingMetadata) {
	if metadata == nil {
		panic("metadata cannot be nil")
	}
	metadata.SetName(string(*n))
}

func (n *Named) Matches(metadata *BindingMetadata) bool {
	if metadata == nil {
		panic("metadata cannot be nil")
	}
	name  := metadata.Name()
	return name == "" || metadata.Name() == string(*n)
}

// Metadata matches against kev/value pairs.
type Metadata struct {
	metadata KeyValues
}

func (m *Metadata) InitWithMetadata(
	metadata KeyValues,
) error {
	if metadata == nil {
		panic("metadata cannot be nil")
	}
	if m.metadata != nil {
		panic("Metadata already initialized")
	}
	m.metadata = make(KeyValues)
	for key, value := range metadata {
		m.metadata[key] = value
	}
	return nil
}

func (m *Metadata) InitWithTag(
	tag reflect.StructTag,
) (err error) {
	if m.metadata != nil {
		panic("Metadata already initialized")
	}
	m.metadata = make(KeyValues)
	if tag, ok := tag.Lookup("metadata"); ok {
		if tag == "" {
			return nil
		}
		for _, metadata := range strings.Split(tag, ",") {
			var meta = strings.SplitN(metadata, "=", 2)
			switch len(meta) {
			case 1:
				m.metadata[meta[0]] = nil
			case 2:
				m.metadata[meta[0]] = meta[1]
			default:
				err = multierror.Append(err,
					fmt.Errorf("invalid metadata [%v]", metadata))
			}
		}
	}
	return err
}

func (m *Metadata) Require(metadata *BindingMetadata) {
	if metadata == nil {
		panic("metadata cannot be nil")
	}
	for key, value := range m.metadata {
		metadata.Set(key, value)
	}
}

func (m *Metadata) Matches(metadata *BindingMetadata) bool {
	if metadata == nil {
		panic("metadata cannot be nil")
	}
	for key, value := range m.metadata {
		val, found := metadata.Get(key)
		if !found || value != val {
			return false
		}
	}
	return true
}

// Qualifier matches against a type.
type Qualifier[T any] struct {}

func (q Qualifier[T]) Require(metadata *BindingMetadata) {
	metadata.Set(TypeOf[T](), nil)
}

func (q Qualifier[T]) Matches(metadata *BindingMetadata) bool {
	return metadata.IsEmpty() || metadata.Has(TypeOf[T]())
}

// constraintFilter enforces constraints.
type constraintFilter struct{}

func (c *constraintFilter) Order() int {
	return FilterStage
}

func (c *constraintFilter) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if cp, ok := provider.(interface {
		Constraints() []BindingConstraint
	}); ok {
		if scope, ok := ctx.Callback().(BindingScope); ok {
			metadata := scope.Metadata()
			if metadata != nil {
				for _, c := range cp.Constraints() {
					if !c.Matches(metadata) {
						return next.Abort()
					}
				}
			}
		}
	}
	return next.Pipe()
}

var _constraintFilter = []Filter{&constraintFilter{}}

// ConstraintProvider is a FilterProvider for constraints.
type ConstraintProvider struct {
	constraints []BindingConstraint
}

func (c *ConstraintProvider) Constraints() []BindingConstraint {
	return c.constraints
}

func (c *ConstraintProvider) Required() bool {
	return true
}

func (c *ConstraintProvider) AppliesTo(
	callback Callback,
) bool {
	_, ok := callback.(BindingScope)
	return ok
}

func (c *ConstraintProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return _constraintFilter, nil
}

// ConstraintBuilder is a fluent builder for BindingMetadata.
type ConstraintBuilder struct {
	metadata *BindingMetadata
}

func (b *ConstraintBuilder) Named(
	name string,
) *ConstraintBuilder {
	var n = Named(name)
	return b.WithConstraint(&n)
}

func (b *ConstraintBuilder) WithConstraint(
	constraint BindingConstraint,
) *ConstraintBuilder {
	if IsNil(constraint) {
		panic("key cannot be nil")
	}
	constraint.Require(b.Metadata())
	return b
}

func (b *ConstraintBuilder) WithMetadata(
	metadata BindingMetadata,
) *ConstraintBuilder {
	m := b.Metadata()
	name := metadata.Name()
	if len(name) > 0 {
		m.SetName(name)
	}
	metadata.MergeInto(m)
	return b
}

func (b *ConstraintBuilder) Metadata() (metadata *BindingMetadata) {
	if metadata = b.metadata; metadata == nil {
		metadata = new(BindingMetadata)
		b.metadata = metadata
	}
	return metadata
}

func WithName(name string) func(*ConstraintBuilder) {
	return func(c *ConstraintBuilder) {
		c.Named(name)
	}
}

func WithConstraint(constraint BindingConstraint) func(*ConstraintBuilder) {
	return func(c *ConstraintBuilder) {
		c.WithConstraint(constraint)
	}
}

func WithMetadata(metadata BindingMetadata) func(*ConstraintBuilder) {
	return func(c *ConstraintBuilder) {
		c.WithMetadata(metadata)
	}
}

func WithQualifier[T any]() func(*ConstraintBuilder) {
	return func(c *ConstraintBuilder) {
		c.WithConstraint(Qualifier[T]{})
	}
}

// ConstraintInitialize defines generic contract for providers.
type ConstraintInitialize interface {
	BindingConstraint
	Init() error
}

func WithConstraintProvider[T ConstraintInitialize](provider T) func(*ConstraintBuilder) {
	return func(c *ConstraintBuilder) {
		if err := provider.Init(); err != nil {
			panic(fmt.Errorf("invalid provider: %w", err))
		}
		c.WithConstraint(provider)
	}
}

func ApplyConstraints(
	scope       BindingScope,
	constraints ... func(*ConstraintBuilder),
) *BindingMetadata {
	if IsNil(scope) {
		panic("scope cannot be nil")
	}
	if metadata := scope.Metadata(); metadata != nil {
		builder := ConstraintBuilder{metadata}
		for _, constraint := range constraints {
			if constraint != nil {
				constraint(&builder)
			}
		}
		return builder.Metadata()
	}
	return nil
}

var (
	ErrNameMissing = errors.New("the Named constraint requires a non-empty `name:[name]` tag")
)