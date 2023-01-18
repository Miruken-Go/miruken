package govalidator

import (
	"errors"
	"github.com/asaskevich/govalidator"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
)

type validator struct{}

func (v *validator) Validate(
	validates *validate.Validates, target any,
) miruken.HandleResult {
	if !miruken.IsStruct(target) {
		return miruken.NotHandled
	}
	if result, err := govalidator.ValidateStruct(target); !result {
		switch e := err.(type) {
		case govalidator.Errors:
			v.buildValidationOutcome(validates.Outcome(), e)
		default:
			validates.Outcome().
				AddError("", errors.New("failed validation"))
		}
		return miruken.HandledAndStop
	}
	return miruken.Handled
}

func  (v *validator) buildValidationOutcome(
	outcome  *validate.Outcome,
	errors   govalidator.Errors,
) {
	for _, err := range errors {
		switch actual := err.(type) {
		case govalidator.Error:
			pathOutcome(outcome, actual).AddError(actual.Name, actual)
		case govalidator.Errors:
			v.buildValidationOutcome(outcome, actual)
		default:
			outcome.AddError("", err)
		}
	}
}

func pathOutcome(
	outcome  *validate.Outcome,
	err      govalidator.Error,
) *validate.Outcome {
	if path := err.Path; len(path) > 0 {
		for _, field := range path {
			outcome = outcome.RequirePath(field)
		}
	}
	return outcome
}