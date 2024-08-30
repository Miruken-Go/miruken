package event

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal"
)

// Applies events invariantly to an aggregate.
type Applies struct {
	miruken.CallbackBase
	event any
}

func (a *Applies) Source() any {
	return a.event
}

func (a *Applies) Key() any {
	return reflect.TypeOf(a.event)
}

func (a *Applies) Policy() miruken.Policy {
	return appliesPolicyIns
}

func (a *Applies) CanInfer() bool {
	return false
}

func (a *Applies) CanFilter() bool {
	return false
}

func (a *Applies) CanBatch() bool {
	return false
}

func (a *Applies) Dispatch(
	handler  any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	return miruken.DispatchPolicy(handler, a, greedy, composer)
}

func (a *Applies) String() string {
	return fmt.Sprintf("applies => %v", a.event)
}

// AppliesBuilder builds Applies events.
type AppliesBuilder struct {
	miruken.CallbackBuilder
	event any
}

func (b *AppliesBuilder) WithEvent(
	event any,
) *AppliesBuilder {
	if internal.IsNil(event) {
		panic("event cannot be nil")
	}
	b.event = event
	return b
}

func (b *AppliesBuilder) New() *Applies {
	return &Applies{
		CallbackBase: b.CallbackBase(),
		event:        b.event,
	}
}

// Apply applies an event with no results.
// returns an empty promise if execution is asynchronous.
func Apply(
	handler miruken.Handler,
	event   any,
) error {
	if internal.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder AppliesBuilder
	applies := builder.WithEvent(event).New()
	if result := handler.Handle(applies, false, nil); result.IsError() {
		return result.Error()
	} else if !result.Handled() {
		return &miruken.NotHandledError{Callback: event}
	}
	return nil
}

var appliesPolicyIns miruken.Policy = &miruken.InvariantPolicy{}
