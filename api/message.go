package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)

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