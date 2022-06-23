package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)

// Post sends a message without an expected response.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Post(
	handler miruken.Handler,
	message any,
) (*promise.Promise[miruken.Void], error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	return miruken.Command(stash, message)
}

// Send sends a request with an expected response.
// A new Stash is created to manage any transit state.
// Returns the TResponse if the call is synchronous or
// return a promise to TResponse if the call asynchronous.
func Send[TResponse any](
	handler miruken.Handler,
	request any,
) (TResponse, *promise.Promise[TResponse], error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(request) {
		panic("request cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	return miruken.Execute[TResponse](stash, request)
}

// Publish sends a message to all recipients.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Publish(
	handler miruken.Handler,
	message any,
) (*promise.Promise[miruken.Void], error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	if pv, err := miruken.CommandAll(stash, message); err == nil {
		return pv, err
	} else if _, ok := err.(miruken.NotHandledError); ok {
		return nil, nil
	} else {
		return pv, err
	}
}