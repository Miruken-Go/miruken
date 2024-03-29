package govalidator

import (
	"github.com/miruken-go/miruken/setup"
	"github.com/miruken-go/miruken/validates"
)

// Installer integrates validation support for the go validator.
// https://github.com/asaskevich/govalidator
type Installer struct{}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{validates.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&validator{})
	}
	return nil
}

func Feature(config ...func(*Installer)) setup.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
