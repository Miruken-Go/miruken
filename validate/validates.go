package validate

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

// Validates callbacks contravariantly.
type Validates struct {
	miruken.CallbackBase
	source  any
	groups  []any
	outcome Outcome
}

func (v *Validates) Source() any {
	return v.source
}

func (v *Validates) Groups() []any {
	return v.groups
}

func (v *Validates) InGroup(group any) bool {
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

func (v *Validates) Outcome() *Outcome {
	return &v.outcome
}

func (v *Validates) Key() any {
	return reflect.TypeOf(v.source)
}

func (v *Validates) Policy() miruken.Policy {
	return _policy
}

func (v *Validates) Dispatch(
	handler  any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	return miruken.DispatchPolicy(handler, v, greedy, composer)
}

func (v *Validates) String() string {
	return fmt.Sprintf("Validates => %+v", v.source)
}


// Group marks a set of related validations.
type Group struct {
	groups map[any]miruken.Void
}

func (g *Group) Required() bool {
	return true
}

func (g *Group) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		g.groups = make(map[any]miruken.Void)
		if group := strings.TrimSpace(name); len(group) > 0 {
			g.groups[group] = miruken.Void{}
		}
	}
	if len(g.groups) == 0 {
		return errors.New("the Group constraint requires a non-empty `name:group` tag")
	}
	return nil
}

func (g *Group) Merge(constraint miruken.BindingConstraint) bool {
	if group, ok := constraint.(*Group); ok {
		for grp := range group.groups {
			g.groups[grp] = miruken.Void{}
		}
		return true
	}
	return false
}

func (g *Group) Satisfies(required miruken.BindingConstraint) bool {
	rg, ok := required.(*Group)
	if !ok {
		return false
	}
	if _, all := g.groups[_anyGroup]; all {
		return true
	}
	for group := range rg.groups {
		if group == _anyGroup {
			return true
		}
		if _, found := g.groups[group]; found {
			return true
		}
	}
	return false
}

// Rules builds a validation Group constraint.
func Rules(groups ... any) miruken.BindingConstraint {
	if len(groups) == 0 {
		panic("at least one group required")
	}
	groupMap := make(map[any]miruken.Void)
	for _, group := range groups {
		groupMap[group] = miruken.Void{}
	}
	return &Group{groups: groupMap}
}

// Builder builds Validates callbacks.
type Builder struct {
	miruken.CallbackBuilder
	target any
}

func (b *Builder) Target(
	target any,
) *Builder {
	if miruken.IsNil(target) {
		panic("source cannot be nil")
	}
	b.target = target
	return b
}

func (b *Builder) NewValidates() *Validates {
	return &Validates{
		CallbackBase: b.CallbackBase(),
		source:       b.target,
	}
}

// Validate initiates validation of the `target`.
func Validate(
	handler         miruken.Handler,
	target          any,
	constraints ... any,
) (o *Outcome, po *promise.Promise[*Outcome], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.Target(target).
			WithConstraints(constraints...)
	validates := builder.NewValidates()
	if result := handler.Handle(validates, true, nil); result.IsError() {
		err = result.Error()
	} else if !result.IsHandled() {
		o = validates.Outcome()
		setTargetValidationOutcome(target, o)
	} else if _, pv := validates.Result(false); pv == nil {
		o = validates.Outcome()
		setTargetValidationOutcome(target, o)
	} else {
		po = promise.Then(pv, func(any) *Outcome {
			outcome := validates.Outcome()
			setTargetValidationOutcome(target, outcome)
			return outcome
		})
	}
	return
}

func setTargetValidationOutcome(
	target  any,
	outcome *Outcome,
) {
	if v, ok := target.(interface {
		SetValidationOutcome(*Outcome)
	}); ok {
		v.SetValidationOutcome(outcome)
	}
}

var (
	_policy miruken.Policy = &miruken.ContravariantPolicy{}
	_anyGroup              = "*"
)
