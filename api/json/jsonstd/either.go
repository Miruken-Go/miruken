package jsonstd

import (
	"encoding/json"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/maps"
)

// EitherSurrogate is a surrogate for miruken.Either using standard json.
type EitherSurrogate[L, R any] struct {
	Left  bool            `json:"left"`
	Value json.RawMessage `json:"value"`
}

func (s EitherSurrogate[L, R]) Original(
	composer miruken.Handler,
) (any, error) {
	if v, _, err := maps.Map[any](composer, string(s.Value), api.FromJson); err != nil {
		return nil, err
	} else {
		if sur, ok := v.(api.Surrogate); ok {
			if v, err = sur.Original(composer); err != nil {
				return  nil, err
			}
		}
		if s.Left {
			if l, ok := v.(L); ok {
				return either.Left(l), nil
			}
			return nil, fmt.Errorf("expected left of %s", miruken.TypeOf[L]())
		} else {
			if r, ok := v.(R); ok {
				return either.Right(r), nil
			}
			return nil, fmt.Errorf("expected right of %s", miruken.TypeOf[R]())
		}
	}
}