package json

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/validates"
)

// OutcomeSurrogate is a surrogate for validates.Outcome over json.
type OutcomeSurrogate []struct {
	PropertyName string
	Errors       []string
	Nested       OutcomeSurrogate
}

func (s OutcomeSurrogate) Original() any {
	return surrogateToOutcome(s)
}

func (m *SurrogateMapper) ReplaceOutcome(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, outcome *validates.Outcome,
	ctx miruken.HandleContext,
) (string, error) {
	sur := outcomeToSurrogate(outcome)
	js, _, err := maps.Map[string](ctx.Composer(), sur, api.ToJson)
	return js, err
}

func outcomeToSurrogate(outcome *validates.Outcome) OutcomeSurrogate {
	var sur OutcomeSurrogate
	for _, field := range outcome.Fields() {
		var messages []string
		var children OutcomeSurrogate
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
			Nested       OutcomeSurrogate
		}{
			PropertyName: field,
			Errors: 	  messages,
			Nested: 	  children,
		})
	}
	return sur
}

func surrogateToOutcome(surrogate OutcomeSurrogate) *validates.Outcome {
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


// ErrorSurrogate is a surrogate for a generic error over json.
type ErrorSurrogate struct {
	Message string
}

func (s *ErrorSurrogate) Original() any {
	return errors.New(s.Message)
}

func (m *SurrogateMapper) ReplaceError(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, err error,
	ctx miruken.HandleContext,
) (string, error) {
	sur := ErrorSurrogate{err.Error()}
	js, _, err := maps.Map[string](ctx.Composer(), sur, api.ToJson)
	return js, err
}

func (m *SurrogateMapper) New(
	_*struct{
		o creates.It `key:"json.OutcomeSurrogate"`
		e creates.It `key:"json.ErrorSurrogate"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "json.OutcomeSurrogate":
		return new(OutcomeSurrogate)
	case "json.ErrorSurrogate":
		return new(ErrorSurrogate)
	}
	return nil
}
