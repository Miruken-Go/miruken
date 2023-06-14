package security

type (
	// Role represents a certain level of authorization.
	Role struct {
		Name string
	}

	// Group organizes users having common capabilities.
	Group struct {
		Name string
	}

	system struct {}
)

// System defines a singleton principal that can be used
// to bypass security checks.
// e.g. internal service to service interactions
var System = system{}