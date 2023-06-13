package security

// Subject is any entity that requests access to a resource.
// e.g. Process, Machine, Service or User
type Subject interface {
	// Principals returns identities of this Subject
	Principals() []Principal

	// Credentials return security-related attributes of this Subject.
	Credentials() []any
}
