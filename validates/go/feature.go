package govalidator

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validates"
)

// Installer integrates validation support for the go validator.
// https://github.com/asaskevich/govalidator
type Installer struct{}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{validates.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&validator{})
	}
	return nil
}

func Feature(config ...func(*Installer)) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
