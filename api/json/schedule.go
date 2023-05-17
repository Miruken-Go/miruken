package json

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
)

type (
	// Concurrent is a surrogate for api.ConcurrentBatch over json.
	Concurrent []any

	// Sequential is a surrogate for api.SequentialBatch over json.
	Sequential []any
)


// Concurrent

func (c Concurrent) Original(miruken.Handler) (any, error) {
	return &api.ConcurrentBatch{Requests: c}, nil
}


// Sequential

func (s Sequential) Original(miruken.Handler) (any, error) {
	return &api.SequentialBatch{Requests: s}, nil
}


// SurrogateMapper

func (m *SurrogateMapper) ReplaceConcurrent(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, batch api.ConcurrentBatch,
	ctx miruken.HandleContext,
) (byt []byte, err error) {
	sur := Concurrent(batch.Requests)
	if sur == nil {
		sur = make(Concurrent, 0)
	}
	byt, _, _, err = maps.Out[[]byte](ctx.Composer(), sur, api.ToJson)
	return
}

func (m *SurrogateMapper) ReplaceSequential(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, batch api.SequentialBatch,
	ctx miruken.HandleContext,
) (byt []byte, err error) {
	sur := Sequential(batch.Requests)
	if sur == nil {
		sur = make(Sequential, 0)
	}
	byt, _, _, err = maps.Out[[]byte](ctx.Composer(), sur, api.ToJson)
	return
}
