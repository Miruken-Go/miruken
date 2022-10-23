package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/slices"
)

type (
	// Routed wraps a message with route information.
	Routed struct {
		message any
		route   string
	}

	// BatchRouter handles Routed batch requests.
	BatchRouter struct {
		groups map[string][]pending
	}

	// RouteReply holds the responses for a route.
	RouteReply struct {
		Uri       string
		Responses []any
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

// BatchRouter

func (b *BatchRouter) NoConstructor() {}

func (b *BatchRouter) Route(
	_*miruken.Handles, routed Routed,
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return b.batch(routed, ctx.Greedy())
}

func (b *BatchRouter) RouteBatch(
	_*miruken.Handles, routed miruken.Batched[Routed],
	ctx miruken.HandleContext,
) *promise.Promise[any] {
	return b.batch(routed.Source(), ctx.Greedy())
}

func (b *BatchRouter) CompleteBatch(
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

func (b *BatchRouter) batch(
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