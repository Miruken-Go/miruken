package api

import "github.com/miruken-go/miruken"

// Routed wraps a message with route information.
type Routed struct {
	message any
	route   string
}

// Route wraps the message in a Routed container
// with route destination information.
func Route(message any, route string) Routed {
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	if len(route) == 0 {
		panic("route cannot be nil or empty")
	}
	return Routed{message, route}
}