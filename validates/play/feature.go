package playvalidator

import (
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validates"
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
	return []miruken.Feature{validates.Feature()}
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&validator{}).With(v.validate)
		if translator := v.translator; translator != nil {
			setup.With(translator)
		}
	}
	return nil
}

func Feature(config ...func(*Installer)) miruken.Feature {
	installer := &Installer{validate: play.New()}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
