package security

import "github.com/miruken-go/miruken/slices"

// Subject is any entity that requests access to a resource.
// e.g. Process, Machine, Service or User
type (
	Subject interface {
		// Principals return the identities of this Subject.
		// e.g. UserId, Username, Group or Role
		Principals() []any

		// Credentials return security-related attributes of this Subject.
		// e.g. passwords, certificates, claims
		Credentials() []any

		// AddPrincipals adds the principals if not present.
		AddPrincipals(ps ...any)

		// AddCredentials add credentials if not present.
		AddCredentials(cs ...any)
	}

	mutableSubject struct {
		principals  []any
		credentials []any
	}

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

func (s *mutableSubject) AddCredentials(cs ...any) {
	for _, c := range cs {
		if !slices.Contains(s.credentials, c) {
			s.credentials = append(s.credentials, c)
		}
	}
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


// Principals configures a Subject with initial principals.
func Principals(ps ...any) func(Subject) {
	return func(sub Subject) {
		sub.AddPrincipals(ps...)
	}
}

// Credentials configures a Subject with initial credentials.
func Credentials(cs ...any) func(Subject) {
	return func(sub Subject) {
		sub.AddCredentials(cs...)
	}
}

// NewSubject creates a new Subject with optional principals and credentials.
func NewSubject(config ...func(Subject)) Subject {
	sub := &mutableSubject{}
	for _, configure := range config {
		if configure != nil {
			configure(sub)
		}
	}
	return sub
}


var SystemSubject = systemSubject{[]any{System}}