package jwks

import (
	"github.com/miruken-go/miruken/setup"
)

// Installer enables core api support.
type Installer struct {}

func (i *Installer) Install(setup *setup.Builder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&KeySet{})
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
