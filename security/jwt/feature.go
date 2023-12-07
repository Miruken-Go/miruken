package jwt

import (
	"github.com/miruken-go/miruken/security/jwt/jwks"
	"github.com/miruken-go/miruken/setup"
)

// Installer enables jwt authentication.
type Installer struct {}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{jwks.Feature()}
}

func (i *Installer) Install(setup *setup.Builder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&LoginModule{})
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

