package principal

import (
	"fmt"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/security"
)

// StringPrincipal is a generic constraint for string based security.Principal.
type StringPrincipal interface {
	~string
	security.Principal
}

// All returns true if the security.Subject possess all security.Principal's.
func All(subject security.Subject, ps ...security.Principal) bool {
	if internal.IsNil(subject) {
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

// Any returns true if the security.Subject possess any security.Principal's.
func Any(subject security.Subject, ps ...security.Principal) bool {
	if internal.IsNil(subject) {
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

// First returns the first security.Principal of the specified type.
func First[T security.Principal](subject security.Subject) (p T, ok bool) {
	if ps := slices.OfType[security.Principal, T](subject.Principals()); len(ps) > 0 {
		return ps[0], true
	}
	return p, false
}

// Find returns each security.Principal of the specified type.
func Find[T security.Principal](subject security.Subject) []T {
	return slices.OfType[security.Principal, T](subject.Principals())
}


func Parse[T StringPrincipal](val any) []security.Principal {
	switch name := val.(type) {
	case string:
		return []security.Principal{T(name)}
	case []string:
		return slices.Map[string, security.Principal](name,
			func(n string) security.Principal {
				return T(n)
			})
	case []any:
		return slices.Map[any, security.Principal](name,
			func(n any) security.Principal {
				return T(n.(string))
			})
	default:
		panic(fmt.Sprintf("principal: unrecognized value: %v", val))
	}
}
