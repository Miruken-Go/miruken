package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
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
	//var promises *promise.Promise[any]
	//for route, group := range b.groups {
	//
	//}
	return nil, nil, nil
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


// NewRouted wraps the message in a Routed container.
func NewRouted(message any, route string) Routed {
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	if len(route) == 0 {
		panic("route cannot be nil or empty")
	}
	return Routed{message, route}
}