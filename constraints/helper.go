package constraints

import "github.com/miruken-go/miruken"

func First[T miruken.Constraint](
	src miruken.ConstraintSource,
) (first T, ok bool) {
	if miruken.IsNil(src) {
		panic("src cannot be nil")
	}
	for _, constraint := range src.Constraints() {
		if c, ok := constraint.(T); ok {
			return c, true
		}
	}
	return
}