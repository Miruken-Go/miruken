package json

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
)

type (
	// ConcurrentSurrogate is a surrogate for api.ConcurrentBatch over json.
	ConcurrentSurrogate []any

	// SequentialSurrogate is a surrogate for api.SequentialBatch over json.
	SequentialSurrogate []any
)


// ConcurrentSurrogate

func (s ConcurrentSurrogate) Original(miruken.Handler) (any, error) {
	return &api.ConcurrentBatch{Requests: s}, nil
}


// SequentialSurrogate

func (s SequentialSurrogate) Original(miruken.Handler) (any, error) {
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
	sur := ConcurrentSurrogate(batch.Requests)
	if sur == nil {
		sur = make(ConcurrentSurrogate, 0)
	}
	byt, _, err = maps.Map[[]byte](ctx.Composer(), sur, api.ToJson)
	return
}

func (m *SurrogateMapper) ReplaceSequential(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, batch api.SequentialBatch,
	ctx miruken.HandleContext,
) (byt []byte, err error) {
	sur := SequentialSurrogate(batch.Requests)
	if sur == nil {
		sur = make(SequentialSurrogate, 0)
	}
	byt, _, err = maps.Map[[]byte](ctx.Composer(), sur, api.ToJson)
	return
}
