package api

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
)

type (
	// PolymorphicHandling is an enum that determines
	// if messages are augmented with type discriminators.
	PolymorphicHandling uint8

	// PolymorphicOptions provide options for controlling
	// polymorphic messaging.
	PolymorphicOptions struct {
		PolymorphicHandling miruken.Option[PolymorphicHandling]
	}
)

const (
	PolymorphicHandlingNone PolymorphicHandling = 0
	PolymorphicHandlingRoot PolymorphicHandling = 1 << iota
)

// Failure returns a new failed result.
func Failure(val error) either.Either[error, any] {
	return either.Left(val)
}

// Success returns a new successful result.
func Success[R any](val R) either.Either[error, R] {
	return either.Right(val)
}

// Post sends a message without an expected response.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Post(
	handler miruken.Handler,
	message any,
) (p *promise.Promise[miruken.Void], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("post: panic: %v", r)
			}
		}
	}()
	return miruken.Command(stash, message)
}

// Send sends a request with an expected response.
// A new Stash is created to manage any transit state.
// Returns the TResponse if the call is synchronous or
// a promise of TResponse if the call is asynchronous.
func Send[TResponse any](
	handler miruken.Handler,
	request any,
) (r TResponse, pr *promise.Promise[TResponse], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(request) {
		panic("request cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("send: panic: %v", r)
			}
		}
	}()
	return miruken.Execute[TResponse](stash, request)
}

// Publish sends a message to all recipients.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Publish(
	handler miruken.Handler,
	message any,
) (p *promise.Promise[miruken.Void], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("publish: panic: %v", r)
			}
		}
	}()
	if pv, err := miruken.CommandAll(stash, message); err == nil {
		return pv, err
	} else if _, ok := err.(*miruken.NotHandledError); ok {
		return nil, nil
	} else {
		return pv, err
	}
}