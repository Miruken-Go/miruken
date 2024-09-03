package command

import (
	"fmt"
	"reflect"
)

type Metadata string

func (m *Metadata) Name() string {
	return string(*m)
}

func (m *Metadata) InitWithTag(tag reflect.StructTag) error {
	if l, ok := tag.Lookup("command"); ok {
		_, err := fmt.Sscanf(l, "name=%s", m)
		return err
	}
	return nil
}
