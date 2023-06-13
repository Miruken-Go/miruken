package security

type (
	// Principal represents an identity of a Subject.
	// e.g. UserId, Username, Account or Role
	Principal interface {
		// Name returns the name of this Principal.
		Name() string
	}

	// Role represents a certain level of authorization and
	// correspond to one or more privileges in a system.
	Role struct {
		name string
	}
)


func (r Role) Name() string {
	return r.name
}


// NewRole returns a new Role with `name`.
func NewRole(name string) Role {
	return Role{name}
}