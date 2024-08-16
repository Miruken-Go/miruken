package aggregate

import (
	"fmt"
	"reflect"
)

// Export captures aggregate details.
type Export string

//goland:noinspection GoMixedReceiverTypes
func (e Export) Name() string {
	return string(e)
}

//goland:noinspection GoMixedReceiverTypes
func (e *Export) InitWithTag(tag reflect.StructTag) error {
	if aggregate, ok := tag.Lookup("aggregate"); ok {
		_, err := fmt.Sscanf(aggregate, "name=%s", e)
		return err
	}
	return nil
}
