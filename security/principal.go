package security

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/slices"
)

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


// HasAllPrincipals return true if the subject possess all principals.
func HasAllPrincipals(subject Subject, ps ...any) bool {
	if miruken.IsNil(subject) {
		panic("subject cannot be nil")
	}
	sp := subject.Principals()
	for _, p := range ps {
		if !slices.Contains(sp, p) {
			return false
		}
	}
	return true
}

// HasAnyPrincipals return true if the subject possess any principals.
func HasAnyPrincipals(subject Subject, ps ...any) bool {
	if miruken.IsNil(subject) {
		panic("subject cannot be nil")
	}
	sp := subject.Principals()
	for _, p := range ps {
		if slices.Contains(sp, p) {
			return true
		}
	}
	return true
}
