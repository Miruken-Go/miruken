package command

import (
	"fmt"
	"reflect"
)

// Export captures command details.
type Export string

//goland:noinspection GoMixedReceiverTypes
func (e Export) Name() string {
	return string(e)
}

//goland:noinspection GoMixedReceiverTypes
func (e *Export) InitWithTag(tag reflect.StructTag) error {
	if command, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(command, "name=%s", e)
		return err
	}
	return nil
}
