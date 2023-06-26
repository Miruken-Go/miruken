package authorizes

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"reflect"
)

type (
	// It authorizes callbacks contravariantly.
	It struct {
		miruken.CallbackBase
		action  any
	}

	// Options control the authorization process.
 	Options struct {
		RequirePolicy bool
	}
)


func (a *It) Source() any {
	return a.action
}

func (a *It) Key() any {
	return reflect.TypeOf(a.action)
}

func (a *It) Policy() miruken.Policy {
	return policy
}

func (a *It) Dispatch(
	handler  any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	return miruken.DispatchPolicy(handler, a, greedy, composer)
}

func (a *It) String() string {
	return fmt.Sprintf("authorizes %+v", a.action)
}


// Builder builds It callbacks.
type Builder struct {
	miruken.CallbackBuilder
	action  any
	subject security.Subject
}

func (b *Builder) ForAction(
	action any,
) *Builder {
	if miruken.IsNil(action) {
		panic("action cannot be nil")
	}
	b.action = action
	return b
}

func (b *Builder) WithSubject(
	subject security.Subject,
) *Builder {
	if miruken.IsNil(subject) {
		panic("subject cannot be nil")
	}
	b.subject = subject
	return b
}

func (b *Builder) New() *It {
	return &It{
		CallbackBase: b.CallbackBase(),
		action:       b.action,
	}
}

// Access performs authorization on `action`.
func Access(
	handler     miruken.Handler,
	action      any,
	constraints ...any,
) (bool, *promise.Promise[bool], error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(action) {
		panic("action cannot be nil")
	}
	var options Options
	miruken.GetOptions(handler, &options)
	var builder Builder
	builder.ForAction(action).
		    WithConstraints(constraints...)
	auth := builder.New()
	if result := handler.Handle(auth, false, nil); result.IsError() {
		return false, nil, result.Error()
	} else if !result.Handled() {
		return !options.RequirePolicy, nil, nil
	} else if r, pr := auth.Result(false); pr == nil {
		return r == true, nil, nil
	} else {
		return false, promise.Then(pr, func(res any) bool {
			return res == true
		}), nil
	}
}


var policy miruken.Policy = &miruken.ContravariantPolicy{}

