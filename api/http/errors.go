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
	_*miruken.Maps, _ *validate.Outcome,
) int {
	return http.StatusUnprocessableEntity
}