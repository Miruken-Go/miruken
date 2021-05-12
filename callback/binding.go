package callback

import "miruken.com/miruken"

type Binding interface {
	SatisfiesConstraint(
		value    interface{},
		variance miruken.Variance,
	) bool
}

type emptyBinding struct {

}

func (b *emptyBinding) SatisfiesConstraint(
	value    interface{},
	variance miruken.Variance,
) bool {
	return false
}