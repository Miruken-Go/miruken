package authorizes

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
	"reflect"
)

type (
	// It authorizes callbacks contravariantly.
	It struct {
		miruken.CallbackBase
		key     any
		action  any
		subject security.Subject
	}

	// Options control the authorization process.
 	Options struct {
		RequirePolicy bool
	}
)


func (a *It) Action() any {
	return a.action
}

func (a *It) Key() any {
	if key := a.key; reflect.ValueOf(key).IsZero() {
		return reflect.TypeOf(a.action)
	} else {
		return key
	}
}

func (a *It) Subject() security.Subject {
	return a.subject
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
	key     any
	action  any
	subject security.Subject
}

func (b *Builder) WithKey(
	key any,
) *Builder {
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
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
		key:          b.key,
		action:       b.action,
		subject:      b.subject,
	}
}


// Input performs authorization on `action`.
func Input(
	handler     miruken.Handler,
	action      any,
	subject     security.Subject,
	constraints ...any,
) (bool, *promise.Promise[bool], error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(subject) {
		panic("subject cannot be nil")
	}
	var options Options
	miruken.GetOptions(handler, &options)
	var builder Builder
	builder.ForAction(action).
		    WithSubject(subject).
		    WithConstraints(constraints...)
	auth := builder.New()
	handler = miruken.BuildUp(handler, provides.With(subject))
	if result := handler.Handle(auth, true, nil); result.IsError() {
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


var (
	policy miruken.Policy = &miruken.ContravariantPolicy{}
)
