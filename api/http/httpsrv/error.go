package httpsrv

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/validates"
	"net/http"
)

type (
	StatusCodeMapper struct {}
)

func (s *StatusCodeMapper) NotHandled(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	  }, nhe *miruken.NotHandledError,
) int {
	if _, ok := nhe.Callback.(*maps.It); ok {
		return http.StatusUnsupportedMediaType
	}
	return http.StatusInternalServerError
}

func (s *StatusCodeMapper) UnknownTypeId(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	  }, _ *api.UnknownTypeIdError,
) int {
	return http.StatusBadRequest
}

func (s *StatusCodeMapper) Validation(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	  }, _ *validates.Outcome,
) int {
	return http.StatusUnprocessableEntity
}

func (s *StatusCodeMapper) JsonSyntax(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	}, _ *json.SyntaxError,
) int {
	return http.StatusBadRequest
}