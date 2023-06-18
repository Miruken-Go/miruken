package security

import "github.com/miruken-go/miruken/slices"

// Subject is any entity that requests access to a resource.
// e.g. Process, Machine, Service or User
type (
	Subject interface {
		// Principals return the identities of this Subject.
		// e.g. UserId, Username, Group or Role
		Principals() []any

		// Credentials return security attributes of this Subject.
		// e.g. passwords, certificates, claims
		Credentials() []any

		// AddPrincipals adds any new principals to this Subject.
		AddPrincipals(ps ...any)

		// AddCredentials add any new credentials to this Subject.
		AddCredentials(cs ...any)

		// RemovePrincipals remove the principals from this Subject.
		RemovePrincipals(ps ...any)

		// RemoveCredentials remove the credentials from this Subject.
		RemoveCredentials(cs ...any)
	}

	// SubjectOption allows configuration of new Subject.
	SubjectOption func(subject Subject)

	mutableSubject struct {
		principals  []any
		credentials []any
	}

	system struct {}

	systemSubject struct{
		principals []any
	}
)


// Subject

func (s *mutableSubject) Principals() []any {
	return s.principals
}

func (s *mutableSubject) Credentials() []any {
	return s.credentials
}

func (s *mutableSubject) AddPrincipals(ps ...any) {
	for _, p := range ps {
		if !slices.Contains(s.principals, p) {
			s.principals = append(s.principals, p)
		}
	}
}

func (s *mutableSubject) RemovePrincipals(ps ...any) {
	s.principals = slices.Remove(s.principals, ps...)
}

func (s *mutableSubject) AddCredentials(cs ...any) {
	for _, c := range cs {
		if !slices.Contains(s.credentials, c) {
			s.credentials = append(s.credentials, c)
		}
	}
}

func (s *mutableSubject) RemoveCredentials(cs ...any) {
	s.credentials = slices.Remove(s.credentials, cs...)
}


// systemSubject

func (s systemSubject) Principals() []any {
	return s.principals
}

func (s systemSubject) Credentials() []any {
	return nil
}

func (s systemSubject) AddPrincipals(ps ...any) {
	panic("system subject is immutable")
}

func (s systemSubject) AddCredentials(cs ...any) {
	panic("system subject is immutable")
}

func (s systemSubject) removePrincipals(ps ...any) {
	panic("system subject is immutable")
}

func (s systemSubject) RemoveCredentials(cs ...any) {
	panic("system subject is immutable")
}


// WithPrincipals configures a Subject with initial principals.
func WithPrincipals(ps ...any) SubjectOption {
	return func(sub Subject) {
		sub.AddPrincipals(ps...)
	}
}

// WithCredentials configures a Subject with initial credentials.
func WithCredentials(cs ...any) SubjectOption {
	return func(sub Subject) {
		sub.AddCredentials(cs...)
	}
}

// NewSubject creates a new Subject with optional principals and credentials.
func NewSubject(opts ...SubjectOption) Subject {
	sub := &mutableSubject{}
	for _, opt := range opts {
		if opt != nil {
			opt(sub)
		}
	}
	return sub
}


var (
	// System defines a singleton principal that can be used
	// to bypass security checks.
	// e.g. internal service to service interactions
	System = system{}

	// SystemSubject is a singleton Subject used to bypass security.
	SystemSubject = systemSubject{[]any{System}}
)
