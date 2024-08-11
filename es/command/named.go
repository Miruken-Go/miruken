package command

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken"
)

// Named matches against a name.
type Named string

//goland:noinspection GoMixedReceiverTypes
func (n Named) Name() string {
	return string(n)
}

//goland:noinspection GoMixedReceiverTypes
func (n Named) Required() bool {
	return false
}

//goland:noinspection GoMixedReceiverTypes
func (n Named) Implied() bool {
	return false
}

//goland:noinspection GoMixedReceiverTypes
func (n *Named) InitWithTag(tag reflect.StructTag) error {
	if command, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(command, "name=%s", n)
		return err
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (n Named) Satisfies(required miruken.Constraint, ctx miruken.HandleContext) bool {
	rn, ok := required.(Named)
	return ok && n == rn
}
