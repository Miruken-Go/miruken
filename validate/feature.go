package validate

import "github.com/miruken-go/miruken"

// Installer configures validation support.
type Installer struct {
	output bool
}

func (v *Installer) ValidateOutput () {
	v.output = true
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.RegisterHandlers(&ApiMapping{})
		setup.RequireFilters(NewProvider(v.output))
	}
	return nil
}

func Output(installer *Installer) {
	installer.ValidateOutput()
}

func Feature(
	config ... func(installer *Installer),
) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag byte
