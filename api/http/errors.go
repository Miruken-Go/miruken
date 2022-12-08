package http

import (
	"github.com/miruken-go/miruken"
	"net/http"
)

type (
	StatusCodeMapper struct {}
)

func (s *StatusCodeMapper) ValidationError(
	_*miruken.Maps, _ *miruken.ValidationOutcome,
) int {
	return http.StatusUnprocessableEntity
}