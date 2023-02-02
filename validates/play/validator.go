package playvalidator

import (
	"errors"
	"fmt"
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validates"
	"strings"
)

type (
	// Validator provides common validation behavior.
	Validator struct {
		validate   *play.Validate
		translator ut.Translator
	}

	// Rules express the validation behavior explicitly
	// without depending on validation struct tags.
	Rules []struct{ Type any; Rules map[string]string }

	validator struct { Validator }
)

// Validator

func (v *Validator) Constructor(
	validate *play.Validate,
	_ *struct{ miruken.Optional }, translator ut.Translator,
) {
	v.validate   = validate
	v.translator = translator
}

func (v *Validator) ConstructWithRules(
	rules      Rules,
	validate   *play.Validate,
	translator ut.Translator,
) {
	if validate == nil {
		validate = play.New()
	}

	for _, rule := range rules {
		validate.RegisterStructValidationMapRules(rule.Rules, rule.Type)
	}

	v.validate   = validate
	v.translator = translator
}

func (v *Validator) Validate(
	target  any,
	outcome *validates.Outcome,
) miruken.HandleResult {
	if !miruken.IsStruct(target) {
		return miruken.NotHandled
	}
	if err := v.validate.Struct(target); err != nil {
		switch e := err.(type) {
		case *play.InvalidValidationError:
			return miruken.NotHandled.WithError(err)
		case play.ValidationErrors:
			if v.translator == nil {
				v.addErrors(outcome, e)
			} else {
				v.translateErrors(outcome, e)
			}
			return miruken.HandledAndStop
		default:
			panic(fmt.Errorf("unexpected validation error: %w", err))
		}
	}
	return miruken.Handled
}

func (v *Validator) ValidateAndStop(
	target  any,
	outcome *validates.Outcome,
) miruken.HandleResult {
	if result := v.Validate(target, outcome); result.Handled() {
		// Stop the generic validator from validating tags
		return result.Or(miruken.HandledAndStop)
	} else {
		return result
	}
}

func (v *Validator) addErrors(
	outcome     *validates.Outcome,
	fieldErrors play.ValidationErrors,
) {
	for _, err := range fieldErrors {
		var path string
		ns    := err.StructNamespace()
		parts := strings.SplitN(ns, ".", 2)
		if len(parts) > 1 { path = parts[1] }
		outcome.AddError(path, err)
	}
}

func (v *Validator) translateErrors(
	outcome     *validates.Outcome,
	fieldErrors play.ValidationErrors,
) {
	for field, msg := range fieldErrors.Translate(v.translator) {
		var path string
		parts := strings.SplitN(field, ".", 2)
		if len(parts) > 1 { path = parts[1] }
		outcome.AddError(path, errors.New(msg))
	}
}


// validator

func (v *validator) Validate(
	it *validates.It, target any,
) miruken.HandleResult {
	return v.Validator.Validate(target, it.Outcome())
}