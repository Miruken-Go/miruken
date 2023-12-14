package json

import (
	"errors"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/validates"
)

// Outcome is a surrogate for validates.Outcome over json.
type Outcome []struct {
	PropertyName string
	Errors       []string
	Nested       Outcome
}

func (s Outcome) Original(miruken.Handler) (any, error) {
	return surrogateToOutcome(s), nil
}

func (m *SurrogateMapper) ReplaceOutcome(
	_ *struct {
		maps.It
		maps.Format `to:"application/json"`
	}, outcome *validates.Outcome,
	ctx miruken.HandleContext,
) ([]byte, error) {
	sur := outcomeToSurrogate(outcome)
	js, _, _, err := maps.Out[[]byte](ctx, sur, api.ToJson)
	return js, err
}

func outcomeToSurrogate(outcome *validates.Outcome) Outcome {
	var sur Outcome
	for _, field := range outcome.Fields() {
		var messages []string
		var children Outcome
		for _, err := range outcome.FieldErrors(field) {
			if child, ok := err.(*validates.Outcome); ok {
				children = append(children, outcomeToSurrogate(child)...)
			} else {
				messages = append(messages, err.Error())
			}
		}
		sur = append(sur, struct {
			PropertyName string
			Errors       []string
			Nested       Outcome
		}{
			PropertyName: field,
			Errors:       messages,
			Nested:       children,
		})
	}
	return sur
}

func surrogateToOutcome(surrogate Outcome) *validates.Outcome {
	outcome := &validates.Outcome{}
	for _, sur := range surrogate {
		field := sur.PropertyName
		if failures := sur.Errors; len(failures) > 0 {
			for _, msg := range failures {
				outcome.AddError(field, errors.New(msg))
			}
		}
		if nested := sur.Nested; len(nested) > 0 {
			outcome.AddError(field, surrogateToOutcome(nested))
		}
	}
	return outcome
}

// Error is a surrogate for a generic error over json.
type Error struct {
	Message string
}

func (s *Error) Original(miruken.Handler) (any, error) {
	return errors.New(s.Message), nil
}

func (m *SurrogateMapper) ReplaceError(
	_ *struct {
		maps.It
		maps.Format `to:"application/json"`
	}, err error,
	ctx miruken.HandleContext,
) ([]byte, error) {
	sur := Error{err.Error()}
	js, _, _, err := maps.Out[[]byte](ctx, sur, api.ToJson)
	return js, err
}
