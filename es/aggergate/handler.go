package aggergate

type (
	// Handler is a FilterProvider that applies event-sourcing rules.
	Handler struct {
		name string
	}

	// filter interprets the current input as a command acting on an
	// aggregate and the output as a sequence of es to be applied
	// to the aggregate.
	// Expects the argument after the command to be the aggregate.
	filter struct{}
)
