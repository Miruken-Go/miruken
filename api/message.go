package api

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// Message is envelop for polymorphic payloads.
	Message struct {
		Payload any
	}

	// Polymorphism is an enum that defines how type
	// discriminators are included in polymorphic messages.
	Polymorphism uint8

	// Options provide options for controlling api messaging.
	Options struct {
		Polymorphism   miruken.Option[Polymorphism]
		TypeInfoFormat string
	}

	// TypeFieldInfo defines metadata for polymorphic messages.
	TypeFieldInfo struct {
		Field string
		Value string
	}

	// GoTypeFieldInfoMapper provides TypeFieldInfo using package and name.
	GoTypeFieldInfoMapper struct {}

	// InvalidTypeIdError reports an invalid type discriminator.
	InvalidTypeIdError struct {
		TypeId string
		Reason error
	}
)

const (
	PolymorphismNone Polymorphism = 0
	PolymorphismRoot Polymorphism = 1 << iota
)

// GoTypeFieldInfoMapper

func (m *GoTypeFieldInfoMapper) TypeFieldInfo(
	_*struct{
		miruken.Maps
		miruken.Format `to:"type:info"`
	  }, maps *miruken.Maps,
) (TypeFieldInfo, error) {
	typ := reflect.TypeOf(maps.Source())
	return TypeFieldInfo{"@type", typ.String()}, nil
}


// InvalidTypeIdError

func (e *InvalidTypeIdError) Error() string {
	return fmt.Sprintf("invalid type id '%s': %s", e.TypeId, e.Reason.Error())
}

func (e *InvalidTypeIdError) Unwrap() error {
	return e.Reason
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


// ToTypeInfo requests type information for a type.
var ToTypeInfo = miruken.To("type:info")
