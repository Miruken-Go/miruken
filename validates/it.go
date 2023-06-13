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

func (g *Group) Implied() bool {
	return false
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

func (g *Group) Satisfies(required miruken.Constraint, _ miruken.Callback) bool {
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
	source any
}

func (b *Builder) ForSource(
	source any,
) *Builder {
	if miruken.IsNil(source) {
		panic("source cannot be nil")
	}
	b.source = source
	return b
}

func (b *Builder) New() *It {
	return &It{
		CallbackBase: b.CallbackBase(),
		source:       b.source,
	}
}

// Source performs all validations on `source`.
func Source(
	handler     miruken.Handler,
	source      any,
	constraints ...any,
) (o *Outcome, po *promise.Promise[*Outcome], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.ForSource(source).
			WithConstraints(constraints...)
	val := builder.New()
	if result := handler.Handle(val, true, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		o = val.Outcome()
		setValidationOutcome(source, o)
	} else if _, pv := val.Result(false); pv == nil {
		o = val.Outcome()
		setValidationOutcome(source, o)
	} else {
		po = promise.Then(pv, func(any) *Outcome {
			outcome := val.Outcome()
			setValidationOutcome(source, outcome)
			return outcome
		})
	}
	return
}

func setValidationOutcome(
	source  any,
	outcome *Outcome,
) {
	if v, ok := source.(interface {
		SetValidationOutcome(*Outcome)
	}); ok {
		v.SetValidationOutcome(outcome)
	}
}

var (
	policy miruken.Policy = &miruken.ContravariantPolicy{}
	anyGroup              = "*"
)
