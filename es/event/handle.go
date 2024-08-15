package event

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
)

type (
	// Handle marks a handler for event processing.
	Handle struct {
		miruken.BindingGroup
		handles.It
		Export
	}
)
