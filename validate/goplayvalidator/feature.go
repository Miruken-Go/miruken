package goplayvalidator

import (
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
)

// Installer integrates validation support for the go playground validator.
// https://github.com/go-playground/validator/
type Installer struct {
	validate   *play.Validate
	translator ut.Translator
}

func (v *Installer) Validator() *play.Validate {
	return v.validate
}

func (v *Installer) UseTranslator(translator ut.Translator) {
	v.translator = translator
}

func (v *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{validate.Feature()}
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.RegisterHandlers(&validator{})
		setup.AddHandlers(miruken.NewProvider(v.validate))
		if trans := v.translator; trans != nil {
			setup.AddHandlers(miruken.NewProvider(trans))
		}
	}
	return nil
}

func Feature(
	config ... func(installer *Installer),
) miruken.Feature {
	installer := &Installer{validate: play.New()}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag byte
