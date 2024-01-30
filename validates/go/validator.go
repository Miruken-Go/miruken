package govalidator

import (
	"errors"

	"github.com/asaskevich/govalidator"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/validates"
)

type validator struct{}

func (v *validator) Validate(
	it *validates.It, target any,
) miruken.HandleResult {
	if !internal.IsStruct(target) {
		return miruken.NotHandled
	}
	if result, err := govalidator.ValidateStruct(target); !result {
		var e govalidator.Errors
		switch {
		case errors.As(err, &e):
			v.addErrors(it.Outcome(), e)
		default:
			it.Outcome().AddError("", errors.New("failed validation"))
		}
		return miruken.HandledAndStop
	}
	return miruken.Handled
}

func (v *validator) addErrors(
	outcome *validates.Outcome,
	errs    govalidator.Errors,
) {
	for _, err := range errs {
		var e govalidator.Error
		var es govalidator.Errors
		switch {
		case errors.As(err, &e):
			pathOutcome(outcome, e).AddError(e.Name, e)
		case errors.As(err, &es):
			v.addErrors(outcome, es)
		default:
			outcome.AddError("", err)
		}
	}
}

func pathOutcome(
	outcome *validates.Outcome,
	err     govalidator.Error,
) *validates.Outcome {
	if path := err.Path; len(path) > 0 {
		for _, field := range path {
			outcome = outcome.RequirePath(field)
		}
	}
	return outcome
}
