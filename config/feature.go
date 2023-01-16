package config

import (
	"github.com/miruken-go/miruken"
)

// Installer configures configuration support.
type Installer struct {
	provider Provider
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		if provider := v.provider; !miruken.IsNil(provider) {
			setup.RegisterHandlers(&Factory{}).
				  AddHandlers(&Factory{v.provider})
		}
	}
	return nil
}

// Feature creates and configures configuration support
// using the supplied configuration Provider.
func Feature(
	provider Provider,
	config ... func(installer *Installer),
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

var _featureTag byte