package api

import "github.com/miruken-go/miruken"

type (
	// ErrorData is the default error shape.
	ErrorData struct {
		Message string
	}
)

var (
	FromError = miruken.To("api:error")
	ToError   = miruken.From("api:error")
)

