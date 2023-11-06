package stdjson

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/maps"
)

type (
	// ScheduledResult is a surrogate for api.ScheduledResult over json.
	ScheduledResult []Either[error, any]
)


func (s ScheduledResult) Original(composer miruken.Handler) (any, error) {
	responses := make([]either.Monad[error, any], len(s))
	for i, resp := range s {
		if orig, err := resp.Original(composer); err != nil {
			return nil, err
		} else {
			responses[i] = orig
		}
	}
	return &api.ScheduledResult{Responses: responses}, nil
}


// SurrogateMapper

func (m *SurrogateMapper) ReplaceScheduledResult(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, result api.ScheduledResult,
	ctx miruken.HandleContext,
) ([]byte, error) {
	sur := make(ScheduledResult, len(result.Responses))
	for i, resp := range result.Responses {
		err := either.Fold(resp, func(e error) error {
			byt, _, _, err := maps.Out[[]byte](ctx, e, api.ToJson)
			if err == nil {
				sur[i] = Either[error, any]{true, byt}
			}
			return err
		}, func(val any) error {
			byt, _, _, err := maps.Out[[]byte](ctx, val, api.ToJson)
			if err == nil {
				sur[i] = Either[error, any]{false, byt}
			}
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	byt, _, _, err := maps.Out[[]byte](ctx, sur, api.ToJson)
	return byt, err
}