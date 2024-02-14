package cascade

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Callbacks is a miruken.Intent for cascading callbacks.
	Callbacks = miruken.Cascade

	// Messages is a miruken.Intent for cascading api messages.
	Messages struct {
		messages []any
		handler  miruken.Handler
		publish  bool
	}
)

var (
	// Handle is a fluent builder for cascading Callbacks.
	Handle = miruken.CascadeCallbacks
)


// Messages

func (m *Messages) WithHandler(
	handler miruken.Handler,
) *Messages {
	m.handler = handler
	return m
}

func (m *Messages) Apply(
	ctx  miruken.HandleContext,
) (promise.Reflect, error) {
	messages := m.messages
	if len(messages) == 0 {
		return nil, nil
	}

	handler := m.handler
	if internal.IsNil(handler) {
		handler = ctx.Composer
	}

	var promises []*promise.Promise[any]

	for _, message := range messages {
		var pc *promise.Promise[any]
		var err error
		if m.publish {
			pc, err = api.Publish(handler, message)
		} else {
			pc, err = api.Post(handler, message)
		}
		if err != nil {
			return nil, err
		} else if pc != nil {
			promises = append(promises, pc)
		}
	}

	switch len(promises) {
	case 0:
		return nil, nil
	case 1:
		return promises[0], nil
	default:
		return promise.All(nil, promises...), nil
	}
}

// Post is a fluent builder for posting Messages.
func Post(messages ...any) *Messages {
	return &Messages{messages: messages}
}

// Publish is a fluent builder for publishing Messages.
func Publish(messages ...any) *Messages {
	return &Messages{messages: messages, publish: true}
}
