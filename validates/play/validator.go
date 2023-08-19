package playvalidator

import (
	"errors"
	"fmt"
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/validates"
	"reflect"
	"strings"
)

type (
	// TypeRules express the validation constraints for a type
	// without depending on validation struct tags.
	TypeRules struct{
		Type        any
		Constraints map[string]string
	}

	// Rules express the validation constraints for a set of types.
	Rules []TypeRules

	// Validator provides core validation behavior.
	Validator struct {
		validate   *play.Validate
		translator ut.Translator
	}

	// Validates handles validation for a specific type.
	// We duplicate the definitions below to allow multiple
	// validations for a single type.  We take advantage of
	// GO composition to inject a validation handle method.
	// Methods are only composed if it appears once.
	// Therefore, we duplicate the validators and define a
	// unique method i.e. Validate1, Validate2, ...
	Validates[T any] struct {
		Validator
	}

	// Validates1 handles validation for a specific type.
	Validates1[T any] struct {
		Validator
	}

	// Validates2 handles validation for a specific type.
	Validates2[T any] struct {
		Validator
	}

	// Validates3 handles validation for a specific type.
	Validates3[T any] struct {
		Validator
	}

	// Validates4 handles validation for a specific type.
	Validates4[T any] struct {
		Validator
	}

	// Validates5 handles validation for a specific type.
	Validates5[T any] struct {
		Validator
	}

	// validator performs default tag based validation.
	validator struct { Validator }
)


// Validator

func (v *Validator) Constructor(
	validate *play.Validate,
	_*struct{args.Optional}, translator ut.Translator,
) {
	v.validate   = validate
	v.translator = translator
}

func (v *Validator) WithRules(
	rules      Rules,
	configure  func(*play.Validate) error,
	translator ut.Translator,
) error {
	if v.validate != nil {
		panic("validator already initialized")
	}

	validate := play.New()
	if configure != nil {
		if err := configure(validate); err != nil {
			return err
		}
	}

	for _, rule := range rules {
		validate.RegisterStructValidationMapRules(rule.Constraints, rule.Type)
	}

	v.validate   = validate
	v.translator = translator
	return nil
}

func (v *Validator) Validate(
	target  any,
	outcome *validates.Outcome,
) miruken.HandleResult {
	if !internal.IsStruct(target) {
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
	result := v.Validate(target, outcome)
	if result.Handled() {
		// Stop the generic validator from validating tags
		return result.Or(miruken.HandledAndStop)
	}
	return result
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


// Type is a helper function to define the constraints for a type.
func Type[T any](constraints map[string]string) TypeRules {
	typ := reflect.Zero(internal.TypeOf[T]()).Interface()
	return TypeRules{Type: typ, Constraints: constraints}
}


// Validates

func (v *Validates[T]) Validate(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// Validates1

func (v *Validates1[T]) Validate1(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// Validates2

func (v *Validates2[T]) Validate2(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// Validates3

func (v *Validates3[T]) Validate3(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// Validates4

func (v *Validates3[T]) Validate4(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// Validates5

func (v *Validates3[T]) Validate5(
	validate *validates.It, t T,
) miruken.HandleResult {
	return v.ValidateAndStop(t, validate.Outcome())
}


// validator

func (v *validator) Validate(
	validate *validates.It, target any,
) miruken.HandleResult {
	return v.Validator.Validate(target, validate.Outcome())
}
