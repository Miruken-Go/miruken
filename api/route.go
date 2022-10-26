package api

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/slices"
	"reflect"
	"strings"
)

type (
	// Routed wraps a message with route information.
	Routed struct {
		message any
		route   string
	}

	// Routes is a FilterProvider of routesFilter.
	Routes struct {
		schemes []string
	}

	// RouteReply holds the responses for a route.
	RouteReply struct {
		Uri       string
		Responses []any
	}

	// routesFilter coordinates miruken.Callback's participating in a batch.
	routesFilter struct {}

	// batchRouter handles Routed batch requests.
	batchRouter struct {
		groups map[string][]pending
	}

	pending struct {
		message  any
		deferred promise.Deferred[any]
	}
)


// Routed

func (r *Routed) Message() any {
	return r.message
}

func (r *Routed) Route() string {
	return r.route
}


// Routes

func (r *Routes) InitWithTag(tag reflect.StructTag) error {
	if schemes, ok := tag.Lookup("scheme"); ok {
		r.schemes = strings.Split(schemes, ",")
	}
	if len(r.schemes) == 0 {
		return errors.New("the Routes filter requires a non-empty `schemes` tag")
	}
	return nil
}

func (r *Routes) Required() bool {
	return true
}

func (r *Routes) AppliesTo(
	callback miruken.Callback,
) bool {
	if handles, ok := callback.(*miruken.Handles); ok {
		_, ok = handles.Source().(*Routed)
		return ok
	}
	return false
}

func (r *Routes) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return _routesFilter, nil
}

// routesFilter

func (r routesFilter) Order() int {
	return miruken.FilterStageLogging - 1
}

func (r routesFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	if _, ok := provider.(*Routes); ok {

	}
	return nil, nil, nil
}


// batchRouter

func (b *batchRouter) NoConstructor() {}

func (b *batchRouter) Route(
	_*miruken.Handles, routed Routed,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return b.batch(routed, ctx.Greedy())
}

func (b *batchRouter) RouteBatch(
	_*miruken.Handles, routed miruken.Batched[Routed],
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return b.batch(routed.Source(), ctx.Greedy())
}

func (b *batchRouter) CompleteBatch(
	composer miruken.Handler,
) (any, *promise.Promise[any], error) {
	var complete []*promise.Promise[any]
	for route, group := range b.groups {
		uri := route
		messages := slices.Map[pending, any](group, func (p pending) any {
			return p.message
		})
		routeTo := RouteTo(&ConcurrentBatch{messages}, route)
		complete = append(complete, promise.Then(sendBatch(composer, routeTo),
			func(results []either.Either[error, any]) RouteReply {
				responses := make([]any, len(results))
				for i, response := range results {
					responses[i] = either.Fold(response,
						func (err error) any {
							group[i].deferred.Reject(err)
							return err
						},
						func (success any) any {
							group[i].deferred.Resolve(success)
							return success
						})
				}
			return RouteReply{ uri, responses }
		}).Catch(func(err error) error {
			// cancel pending promises when available
			return err
		}))
	}
	return nil, promise.All(complete...).Then(func(data any) any {
		return data
	}), nil
}

func (b *batchRouter) batch(
	routed  Routed,
	publish bool,
) *promise.Promise[any] {
	route := routed.Route()

	var group []pending
	if groups := b.groups; groups != nil {
		group = groups[route]
	} else {
		b.groups = make(map[string][]pending)
	}

	msg := routed.Message()
	if publish {
		msg = Published{msg}
	}
	request := pending{
		message:  msg,
		deferred: promise.Defer[any](),
	}
	group = append(group, request)
	b.groups[route] = group

	return request.deferred.Promise()
}

// RouteTo wraps the message in a Routed container.
func RouteTo(message any, route string) Routed {
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	if len(route) == 0 {
		panic("route cannot be nil or empty")
	}
	return Routed{message, route}
}

var _routesFilter = []miruken.Filter{routesFilter{}}