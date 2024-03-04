package miruken

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/miruken-go/miruken/promise"
)

type (
	// Constraint manages BindingMetadata assertions.
	Constraint interface {
		// Required determines if Constraint must be satisfied.
		// If it is not required, it will accept Callback's without it.
		Required() bool

		// Implied determines if Constraint can be satisfied with just HandleContext.
		Implied() bool

		// Satisfies checks if the required Constraint is satisfied.
		// if implied, nil will be passed for `required` Constraint parameter.
		Satisfies(required Constraint, ctx HandleContext) bool
	}

	// ConstraintSource returns one or more Constraint.
	ConstraintSource interface {
		Constraints() []Constraint
	}

	// ConstraintProvider is a FilterProvider for constraints.
	ConstraintProvider struct {
		constraints []Constraint
	}

	// constraintFilter enforces constraints.
	constraintFilter struct{}
)

// ConstraintProvider

func (c *ConstraintProvider) Constraints() []Constraint {
	return c.constraints
}

func (c *ConstraintProvider) Required() bool {
	return true
}

func (c *ConstraintProvider) Filters(
	binding Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return checkConstraints, nil
}

// Named matches against a name.
type Named string

func (n *Named) Name() string {
	return string(*n)
}

func (n *Named) Required() bool {
	return false
}

func (n *Named) Implied() bool {
	return false
}

func (n *Named) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok && len(strings.TrimSpace(name)) > 0 {
		*n = Named(name)
		return nil
	}
	return ErrConstraintNameMissing
}

func (n *Named) Satisfies(required Constraint, _ HandleContext) bool {
	rn, ok := required.(*Named)
	return ok && *n == *rn
}

type (
	// Metadata matches against kev/value pairs.
	Metadata map[any]any

	metadataOwner interface {
		metadata() *Metadata
	}
)

func (m *Metadata) Required() bool {
	return false
}

func (m *Metadata) Implied() bool {
	return false
}

func (m *Metadata) InitWithTag(
	tag reflect.StructTag,
) (err error) {
	if *m != nil {
		panic("Metadata already initialized")
	}
	*m = make(map[any]any)
	if tag, ok := tag.Lookup("metadata"); ok {
		if tag == "" {
			return nil
		}
		for _, metadata := range strings.Split(tag, ",") {
			var meta = strings.SplitN(metadata, "=", 2)
			switch len(meta) {
			case 1:
				(*m)[meta[0]] = nil
			case 2:
				(*m)[meta[0]] = meta[1]
			default:
				err = errors.Join(err, fmt.Errorf("invalid metadata [%v]", metadata))
			}
		}
	}
	return err
}

func (m *Metadata) Satisfies(required Constraint, _ HandleContext) bool {
	rm, ok := required.(metadataOwner)
	return ok && reflect.DeepEqual(rm.metadata(), m.metadata())
}

func (m *Metadata) metadata() *Metadata {
	return m
}

type (
	// Qualifier matches against a type.
	Qualifier[T any] struct{}

	qualifierOwner[T any] interface {
		qualifier() Qualifier[T]
	}
)

func (q Qualifier[T]) Required() bool {
	return false
}

func (q Qualifier[T]) Implied() bool {
	return false
}

func (q Qualifier[T]) Satisfies(required Constraint, _ HandleContext) bool {
	_, ok := required.(qualifierOwner[T])
	return ok
}

func (q Qualifier[T]) qualifier() Qualifier[T] {
	return q
}

// constraintFilter

func (f constraintFilter) Order() int {
	return FilterStage
}

func (f constraintFilter) Next(
	_        Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	if cp, ok := provider.(ConstraintSource); ok {
		callback := ctx.Callback
		constraints := cp.Constraints()
		required := callback.Constraints()
		switch {
		case len(required) == 0:
			// if no required input constraints
			//   implied receiver constraint must be satisfied or
			//   receiver constraint must not be required
			for _, c := range constraints {
				if c.Implied() {
					if !c.Satisfies(nil, ctx) {
						return next.Abort()
					}
				} else if c.Required() {
					return next.Abort()
				}
			}
		case len(constraints) == 0:
			// reject if required input constraints, but no receiver constraints.
			return next.Abort()
		default:
			var matched map[Constraint]struct{}
		Loop:
			for _, rc := range required {
				for _, c := range constraints {
					if !c.Implied() && c.Satisfies(rc, ctx) {
						if c.Required() {
							if matched == nil {
								matched = make(map[Constraint]struct{})
							}
							matched[c] = struct{}{}
						}
						continue Loop
					}
				}
				return next.Abort()
			}
			// Otherwise, every input constraint must be satisfied by at lease one
			// receiver constraint, and every implied constraint must be satisfied.
			for _, c := range constraints {
				if c.Implied() {
					if !c.Satisfies(nil, ctx) {
						return next.Abort()
					}
				} else if c.Required() {
					if _, ok := matched[c]; !ok {
						return next.Abort()
					}
				}
			}
		}
	}
	return next.Pipe()
}

var (
	ErrConstraintNameMissing = errors.New("the Named constraint requires a non-empty `name:[name]` tag")
	checkConstraints         = []Filter{constraintFilter{}}
)
