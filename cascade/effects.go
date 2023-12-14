package cascade

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Callbacks is a miruken.SideEffect for cascading callbacks.
	Callbacks struct {
		callbacks   []any
		constraints []any
		handler     miruken.Handler
		greedy      bool
	}

	// Messages is a miruken.SideEffect for cascading api messages.
	Messages struct {
		messages []any
		handler  miruken.Handler
		publish  bool
	}
)

// Callbacks

func (c *Callbacks) WithConstraints(
	constraints ...any,
) *Callbacks {
	c.constraints = constraints
	return c
}

func (c *Callbacks) WithHandler(
	handler miruken.Handler,
) *Callbacks {
	c.handler = handler
	return c
}

func (c *Callbacks) Greedy(
	greedy bool,
) *Callbacks {
	c.greedy = greedy
	return c
}

func (c *Callbacks) Apply(
	self miruken.SideEffect,
	ctx miruken.HandleContext,
) (promise.Reflect, error) {
	callbacks := c.callbacks
	if len(callbacks) == 0 {
		return nil, nil
	}

	handler := c.handler
	if internal.IsNil(handler) {
		handler = ctx.Composer
	}

	var promises []*promise.Promise[any]

	for _, callback := range callbacks {
		var pc *promise.Promise[any]
		var err error
		if c.greedy {
			pc, err = miruken.CommandAll(handler, callback, c.constraints...)
		} else {
			pc, err = miruken.Command(handler, callback, c.constraints...)
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
		return promise.All(promises...), nil
	}
}

// Messages

func (m *Messages) WithHandler(
	handler miruken.Handler,
) *Messages {
	m.handler = handler
	return m
}

func (m *Messages) Apply(
	self miruken.SideEffect,
	ctx miruken.HandleContext,
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
		return promise.All(promises...), nil
	}
}

// Handle is a fluent builder for Callbacks.
func Handle(callbacks ...any) *Callbacks {
	return &Callbacks{callbacks: callbacks}
}

// Post is a fluent builder for posting Messages.
func Post(messages ...any) *Messages {
	return &Messages{messages: messages}
}

// Publish is a fluent builder for publishing Messages.
func Publish(messages ...any) *Messages {
	return &Messages{messages: messages, publish: true}
}
