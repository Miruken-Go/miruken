package principal

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/slices"
)

// All return true if the subject possess all principals.
func All(subject security.Subject, ps ...security.Principal) bool {
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

// Any return true if the subject possess any principals.
func Any(subject security.Subject, ps ...security.Principal) bool {
	if miruken.IsNil(subject) {
		panic("subject cannot be nil")
	}
	sp := subject.Principals()
	for _, p := range ps {
		if slices.Contains(sp, p) {
			return true
		}
	}
	return false
}
