package json

var (
	// KnownTypeFields holds the list of json property names
	// that can contain type discriminators.
	KnownTypeFields = []string{"$type", "@type"}
)
