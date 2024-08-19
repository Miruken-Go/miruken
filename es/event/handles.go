package event

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
)

type (
	// Handles marks a handler for event processing.
	Handles struct {
		miruken.BindingGroup
		handles.It
		Model
	}
)
