package json

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
)

// ConcurrentSurrogate is a surrogate for api.ConcurrentBatch over json.
type ConcurrentSurrogate []any

func (s ConcurrentSurrogate) Original() any {
	return &api.ConcurrentBatch{Requests: s}
}

func (m *SurrogateMapper) ReplaceConcurrent(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, batch api.ConcurrentBatch,
	ctx miruken.HandleContext,
) (string, error) {
	sur := ConcurrentSurrogate(batch.Requests)
	js, _, err := maps.Map[string](ctx.Composer(), sur, api.ToJson)
	return js, err
}
