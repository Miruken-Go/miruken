package authorizes

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/principal"
)

type (
	// Required is a FilterProvider for authorization.
	Required struct {
		policy any
	}

	// AccessDeniedError indicates authorization failed.
	AccessDeniedError struct {
		Action any
	}

	// filter controls access to actions using policies
	// satisfied by the privileges of a security.Subject.
	filter struct{ miruken.FilterAdapter }
)

// Required

func (r *Required) InitWithTag(tag reflect.StructTag) error {
	if policy, ok := tag.Lookup("policy"); ok {
		r.policy = policy
	}
	return nil
}

func (r *Required) Policy() any {
	return r.policy
}

func (r *Required) Required() bool {
	return true
}

func (r *Required) AppliesTo(
	callback miruken.Callback,
) bool {
	_, ok := callback.(*handles.It)
	return ok
}

func (r *Required) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return filters, nil
}

// AccessDeniedError

func (e *AccessDeniedError) Error() string {
	return fmt.Sprintf("access denied: \"%T\"", e.Action)
}

// filter

func (f filter) Order() int {
	return miruken.FilterStageAuthorization
}

func (f filter) Authorize(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
	subject  security.Subject,
) (out []any, pout *promise.Promise[[]any], err error) {
	if ap, ok := provider.(*Required); ok {
		// System skips checks
		if principal.All(subject, security.System) {
			return next.Pipe()
		}
		callback := ctx.Callback
		composer := ctx.Composer
		action := callback.Source()
		// check binding principals
		if !checkBindingPrincipals(ctx.Binding, subject) {
			return nil, nil, &AccessDeniedError{action}
		}
		// perform authorization check
		g, pg, err := Access(composer, action, ap.policy)
		if err != nil {
			// error performing authorization
			return nil, nil, err
		}
		if pg == nil {
			// if denied return AccessDeniedError.
			if !g {
				return nil, nil, &AccessDeniedError{action}
			}
			// perform the next step in the pipeline
			return next.Pipe()
		}
		// asynchronous authorization check
		return nil, promise.Then(pg, func(g bool) []any {
			// if denied return AccessDeniedError.
			if !g {
				panic(&AccessDeniedError{action})
			}
			return next.PipeAwait()
		}), nil
	}
	return next.Abort()
}

func checkBindingPrincipals(
	binding miruken.Binding,
	subject security.Subject,
) bool {
	for _, m := range binding.Metadata() {
		if p, ok := m.(security.Principal); ok {
			if !principal.All(subject, p) {
				return false
			}
		}
	}
	return true
}

var filters = []miruken.Filter{filter{}}
