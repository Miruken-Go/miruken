package api

import (
	"github.com/miruken-go/miruken/maps"
)

type (
	// ErrorData is the default error shape.
	ErrorData struct {
		Message string
	}
)

var (
	FromError = maps.To("api:error")
	ToError   = maps.From("api:error")
)

