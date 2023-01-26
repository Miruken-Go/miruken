package govalidator

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
)

// Installer integrates validation support for the go validator.
// https://github.com/asaskevich/govalidator
type Installer struct{}

func (v *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{validate.Feature()}
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&featureTag) {
		setup.Specs(&validator{})
	}
	return nil
}

func Feature(
	config ...func(installer *Installer),
) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
