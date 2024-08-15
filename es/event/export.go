package event

import (
	"fmt"
	"reflect"
)

// Export captures event details.
type Export string

//goland:noinspection GoMixedReceiverTypes
func (e Export) Name() string {
	return string(e)
}

//goland:noinspection GoMixedReceiverTypes
func (e *Export) InitWithTag(tag reflect.StructTag) error {
	if event, ok := tag.Lookup("event"); ok {
		_, err := fmt.Sscanf(event, "name=%s", e)
		return err
	}
	return nil
}
