package jsonstd

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/slices"
	"github.com/miruken-go/miruken/validates"
	"github.com/pkg/errors"
)

type (
	// SurrogateMapper maps concepts to json suitable types.
	SurrogateMapper struct {}

	// OutcomeSurrogate is a surrogate for passing validates.Outcome as json.
	OutcomeSurrogate struct {
		PropertyName string
		Errors       []string
		Nested       []OutcomeSurrogate
	}

	// ErrorSurrogate is a surrogate for passing errors over an api.
	ErrorSurrogate struct {
		Message string
	}
)


func (e *ErrorSurrogate) Error() error {
	return errors.New(e.Message)
}

func (m *SurrogateMapper) EncodeOutcome(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, outcome *validates.Outcome,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	ctx miruken.HandleContext,
) (js string, err error) {
	var data []byte
	sur := buildSurrogate(outcome)
	if sur == nil {
		sur = []OutcomeSurrogate{}
	}
	var src any = sur
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		src = &typeContainer{
			v:        src,
			typInfo:  apiOptions.TypeInfoFormat,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
	} else if trans := options.Transformers; len(trans) > 0 {
		src = &transformer{src, trans}
	}
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(src, prefix, indent)
	} else {
		data, err = json.Marshal(src)
	}
	return string(data), err
}

func (m *SurrogateMapper) DecodeOutcome(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, jsonString string,
) (*validates.Outcome, error) {
	var sur []*OutcomeSurrogate
	err := json.Unmarshal([]byte(jsonString), &sur)
	if err != nil {
		return nil, err
	}
	return buildOutcome(slices.Map[*OutcomeSurrogate, OutcomeSurrogate](sur,
		func(s *OutcomeSurrogate) OutcomeSurrogate {
			return *s
		})), nil
}

func (m *SurrogateMapper) EncodeError(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, err error,
) (string, error) {
	sur := &ErrorSurrogate{err.Error()}
	data, err := json.Marshal(sur)
	return string(data), err
}

func (m *SurrogateMapper) New(
	_*struct{
		o creates.It `key:"jsonstd.OutcomeSurrogate"`
	    e creates.It `key:"jsonstd.ErrorSurrogate"`
	  }, create *creates.It,
) (any, error) {
	switch create.Key() {
	case "jsonstd.OutcomeSurrogate":
		return new(OutcomeSurrogate), nil
	case "jsonstd.ErrorSurrogate":
		return new(ErrorSurrogate), nil
	}
	return nil, nil
}

func buildSurrogate(outcome *validates.Outcome) []OutcomeSurrogate {
	return slices.Map[string, OutcomeSurrogate](
		outcome.Fields(),
		func(field string) OutcomeSurrogate {
			var messages []string
			var children []OutcomeSurrogate
			for _, err := range outcome.FieldErrors(field) {
				if child, ok := err.(*validates.Outcome); ok {
					children = append(children, buildSurrogate(child)...)
				} else {
					messages = append(messages, err.Error())
				}
			}
			return OutcomeSurrogate{field, messages, children}
		})
}

func buildOutcome(surrogates []OutcomeSurrogate) *validates.Outcome {
	outcome := &validates.Outcome{}
	for _, sur := range surrogates {
		field := sur.PropertyName
		if failures := sur.Errors; len(failures) > 0 {
			for _, msg := range failures {
				outcome.AddError(field, errors.New(msg))
			}
		}
		if nested := sur.Nested; len(nested) > 0 {
			outcome.AddError(field, buildOutcome(nested))
		}
	}
	return outcome
}
