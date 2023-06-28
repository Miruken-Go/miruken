package login

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security"
)

// Module is responsible for implementing an authentication strategy.
// e.g. basic username/password, oauth token
type Module interface {
	// Login authenticates the subject.
	// It can use the supplied handler to prompt for
	// information such as username or password.
	Login(
		subject security.Subject,
		handler miruken.Handler,
	) error

	// Logout logs out the subject by remove principals
	// and/or credentials from the subject.
	Logout(
		subject security.Subject,
		handler miruken.Handler,
	) error
}