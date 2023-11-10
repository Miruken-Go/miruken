package jwt

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/jwt/jwks"
)

// Installer enables jwt authentication.
type Installer struct {}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{jwks.Feature()}
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&LoginModule{})
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

