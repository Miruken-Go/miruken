package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
	"net/http"
)

type (
	StatusCodeMapper struct {}
)

func (s *StatusCodeMapper) NotHandledError(
	_*struct{
		miruken.Maps
		miruken.Format `to:"http:status-code"`
	  }, nhe *miruken.NotHandledError,
) int {
	if _, ok := nhe.Callback.(*miruken.Maps); ok {
		return http.StatusUnsupportedMediaType
	}
	return http.StatusInternalServerError
}

func (s *StatusCodeMapper) ValidationError(
	_*struct{
		miruken.Maps
		miruken.Format `to:"http:status-code"`
	  }, _ *validate.Outcome,
) int {
	return http.StatusUnprocessableEntity
}