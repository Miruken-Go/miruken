package authorizes

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"reflect"
)

type (
	// Access is a FilterProvider for authorization.
	Access struct {
		policy string
	}

	// filter authorizes input against the current security.Subject.
	filter struct {}
)


// Access

func (a *Access) InitWithTag(tag reflect.StructTag) error {
	if policy, ok := tag.Lookup("policy"); ok {
		a.policy = policy
	}
	return nil
}

func (a *Access) Required() bool {
	return true
}

func (a *Access) AppliesTo(
	callback miruken.Callback,
) bool {
	_, ok := callback.(*handles.It)
	return ok
}

func (a *Access) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return filters, nil
}


// filter

func (f filter) Order() int {
	return miruken.FilterStageAuthorization
}

func (f filter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	return miruken.DynNext(f, next, ctx, provider)
}

func (f filter) DynNext(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
	_*struct{args.Optional}, subject security.Subject,
)  ([]any, *promise.Promise[[]any], error) {
	if _, ok := provider.(*Access); ok {

	}
	return next.Abort()
}

var filters = []miruken.Filter{filter{}}