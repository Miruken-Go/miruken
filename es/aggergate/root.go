package aggergate

import (
	"fmt"
	"reflect"
)

type (
	// Root defines metadata for an aggregate root.
	Root struct {
		name string
	}
)


// Root

func (r *Root) Name() string {
	return r.name
}

func (r *Root) InitWithTag(tag reflect.StructTag) error {
	if agg, ok := tag.Lookup("agg"); ok {
		_, err := fmt.Sscanf(agg, "name=%s", &r.name)
		return err
	}
	return nil
}
