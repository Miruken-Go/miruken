package playvalidator

import (
	"errors"
	"fmt"
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
	"strings"
)

type (
	// Base provides common validation behavior.
	Base struct {
		validate   *play.Validate
		translator ut.Translator
	}

	// Rules express the validation behavior explicitly
	// without depending on validation struct tags.
	Rules []struct{ Type any; Rules map[string]string }

	validator struct { Base }
)

// Base

func (v *Base) Constructor(
	validate *play.Validate,
	_ *struct{ miruken.Optional }, translator ut.Translator,
) {
	v.validate   = validate
	v.translator = translator
}

func (v *Base) ConstructWithRules(
	rules      Rules,
	translator ut.Translator,
) {
	if rules == nil {
		panic("rules cannot be nil")
	}

	val := play.New()
	for _, rule := range rules {
		val.RegisterStructValidationMapRules(rule.Rules, rule.Type)
	}
	v.validate   = val
	v.translator = translator
}

func (v *Base) Validate(
	target  any,
	outcome *validate.Outcome,
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

func (v *Base) ValidateAndStop(
	target  any,
	outcome *validate.Outcome,
) miruken.HandleResult {
	if result := v.Validate(target, outcome); result.Handled() {
		// Stop the generic validator from validating tags
		return result.Or(miruken.HandledAndStop)
	} else {
		return result
	}
}

func (v *Base) addErrors(
	outcome     *validate.Outcome,
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

func (v *Base) translateErrors(
	outcome     *validate.Outcome,
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
	validates *validate.Validates, target any,
) miruken.HandleResult {
	return v.Base.Validate(target, validates.Outcome())
}