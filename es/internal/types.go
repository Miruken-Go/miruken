package internal

import (
	"reflect"

	"github.com/google/uuid"
)

var (
	UUIDType = reflect.TypeFor[uuid.UUID]()
)
