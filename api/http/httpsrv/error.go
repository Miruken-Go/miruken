package httpsrv

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/authorizes"
	"github.com/miruken-go/miruken/validates"
	"net/http"
)

// StatusCodeMapper maps errors into a corresponding http status code.
type StatusCodeMapper struct {}

func (s *StatusCodeMapper) NotHandled(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	  }, _ *miruken.NotHandledError,
) int {
	return http.StatusNotFound
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

func (s *StatusCodeMapper) AccessDenied(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	  }, _ *authorizes.AccessDeniedError,
	_*struct{args.Optional}, subject security.Subject,
) int {
	if miruken.IsNil(subject) || !subject.Authenticated() {
		return http.StatusUnauthorized
	}
	return http.StatusForbidden
}

func (s *StatusCodeMapper) JsonSyntax(
	_*struct{
		maps.It
		maps.Format `to:"http:status-code"`
	}, _ *json.SyntaxError,
) int {
	return http.StatusBadRequest
}