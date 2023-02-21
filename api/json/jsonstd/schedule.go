package jsonstd

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/maps"
)

type (
	// ScheduledResultSurrogate is a surrogate for api.ScheduledResult over json.
	ScheduledResultSurrogate []EitherSurrogate[error, any]
)


func (s ScheduledResultSurrogate) Original(composer miruken.Handler) (any, error) {
	responses := make([]either.Either[error, any], len(s))
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
) (string, error) {
	sur := make(ScheduledResultSurrogate, len(result.Responses))
	for i, resp := range result.Responses {
		err := either.Fold(resp, func(e error) error {
			ej, _, err := maps.Map[string](ctx.Composer(), e, api.ToJson)
			if err == nil {
				sur[i] = EitherSurrogate[error, any]{true, []byte(ej)}
			}
			return err
		}, func(val any) error {
			vj, _, err := maps.Map[string](ctx.Composer(), val, api.ToJson)
			if err == nil {
				sur[i] = EitherSurrogate[error, any]{false, []byte(vj)}
			}
			return err
		})
		if err != nil {
			return "", err
		}
	}
	js, _, err := maps.Map[string](ctx.Composer(), sur, api.ToJson)
	return js, err
}