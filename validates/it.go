package validates

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

type (
	// It validates callbacks contravariantly.
	It struct {
		miruken.CallbackBase
		source  any
		groups  []any
		outcome Outcome
	}

	// Strict alias for validation
	Strict = miruken.Strict
)

func (v *It) Source() any {
	return v.source
}

func (v *It) Groups() []any {
	return v.groups
}

func (v *It) InGroup(group any) bool {
	if len(v.groups) == 0 {
		return false
	}
	for _, grp := range v.groups {
		if grp == group {
			return true
		}
	}
	return false
}

func (v *It) Outcome() *Outcome {
	return &v.outcome
}

func (v *It) Key() any {
	return reflect.TypeOf(v.source)
}

func (v *It) Policy() miruken.Policy {
	return policy
}

func (v *It) Dispatch(
	handler  any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	return miruken.DispatchPolicy(handler, v, greedy, composer)
}

func (v *It) String() string {
	return fmt.Sprintf("validates %+v", v.source)
}


// Group marks a set of related validations.
type Group struct {
	groups map[any]struct{}
}

func (g *Group) Required() bool {
	return true
}

func (g *Group) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		g.groups = make(map[any]struct{})
		if group := strings.TrimSpace(name); len(group) > 0 {
			g.groups[group] = struct{}{}
		}
	}
	if len(g.groups) == 0 {
		return errors.New("the Group constraint requires a non-empty `name:group` tag")
	}
	return nil
}

func (g *Group) Merge(constraint miruken.Constraint) bool {
	if group, ok := constraint.(*Group); ok {
		for grp := range group.groups {
			g.groups[grp] = struct{}{}
		}
		return true
	}
	return false
}

func (g *Group) Satisfies(required miruken.Constraint) bool {
	rg, ok := required.(*Group)
	if !ok {
		return false
	}
	if _, all := g.groups[anyGroup]; all {
		return true
	}
	for group := range rg.groups {
		if group == anyGroup {
			return true
		}
		if _, found := g.groups[group]; found {
			return true
		}
	}
	return false
}

// Groups builds a validation Group constraint.
func Groups(groups ...any) miruken.Constraint {
	if len(groups) == 0 {
		panic("at least one group required")
	}
	groupMap := make(map[any]struct{})
	for _, group := range groups {
		groupMap[group] = struct{}{}
	}
	return &Group{groups: groupMap}
}

// Builder builds It callbacks.
type Builder struct {
	miruken.CallbackBuilder
	src any
}

func (b *Builder) Source(
	src any,
) *Builder {
	if miruken.IsNil(src) {
		panic("source cannot be nil")
	}
	b.src = src
	return b
}

func (b *Builder) New() *It {
	return &It{
		CallbackBase: b.CallbackBase(),
		source:       b.src,
	}
}

// Source performs all validations on `src`.
func Source(
	handler     miruken.Handler,
	src         any,
	constraints ...any,
) (o *Outcome, po *promise.Promise[*Outcome], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.Source(src).
			WithConstraints(constraints...)
	validates := builder.New()
	if result := handler.Handle(validates, true, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		o = validates.Outcome()
		setValidationOutcome(src, o)
	} else if _, pv := validates.Result(false); pv == nil {
		o = validates.Outcome()
		setValidationOutcome(src, o)
	} else {
		po = promise.Then(pv, func(any) *Outcome {
			outcome := validates.Outcome()
			setValidationOutcome(src, outcome)
			return outcome
		})
	}
	return
}

func setValidationOutcome(
	src     any,
	outcome *Outcome,
) {
	if v, ok := src.(interface {
		SetValidationOutcome(*Outcome)
	}); ok {
		v.SetValidationOutcome(outcome)
	}
}

var (
	policy miruken.Policy = &miruken.ContravariantPolicy{}
	anyGroup              = "*"
)
