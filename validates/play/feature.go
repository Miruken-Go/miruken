package playvalidator

import (
	ut "github.com/go-playground/universal-translator"
	play "github.com/go-playground/validator/v10"
	"github.com/miruken-go/miruken/setup"
	"github.com/miruken-go/miruken/validates"
)

// Installer integrates validation support for the go playground validator.
// https://github.com/go-playground/validator/
type Installer struct {
	validate   *play.Validate
	translator ut.Translator
}

func (i *Installer) Validator() *play.Validate {
	return i.validate
}

func (i *Installer) UseTranslator(translator ut.Translator) {
	i.translator = translator
}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{validates.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&validator{}).With(i.validate)
		if translator := i.translator; translator != nil {
			b.With(translator)
		}
	}
	return nil
}

func Feature(config ...func(*Installer)) setup.Feature {
	installer := &Installer{validate: play.New()}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
