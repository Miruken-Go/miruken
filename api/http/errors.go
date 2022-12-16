package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
	"net/http"
)

type (
	StatusCodeMapper struct {}
)

func (s *StatusCodeMapper) ValidationError(
	_*struct{
		miruken.Maps
		miruken.Format `to:"http:status-code"`
	  }, _ *validate.Outcome,
) int {
	return http.StatusUnprocessableEntity
}