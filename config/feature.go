package config

import (
	"github.com/miruken-go/miruken"
)

type (
	// Provider defines the api to allow configuration
	// providers to expose their configuration information.
	Provider interface {
		Unmarshal(path string, flat bool, output any) error
	}

	// Installer enables configuration support.
	Installer struct {
		provider Provider
	}
)

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		if provider := v.provider; !miruken.IsNil(provider) {
			setup.Specs(&Factory{}).
				  Handlers(&Factory{v.provider})
		}
	}
	return nil
}

// Feature creates and configures configuration support
// using the supplied configuration Provider.
func Feature(
	provider Provider,
	config ...func(*Installer),
) miruken.Feature {
	if miruken.IsNil(provider) {
		panic("provider cannot be nil")
	}
	installer := &Installer{provider}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte