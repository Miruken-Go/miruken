package api

import "github.com/miruken-go/miruken"

// Installer enables api support.
type Installer struct {}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.RegisterHandlers(
			&Stash{},
			&Scheduler{},
			&PassThroughRouter{},
			&batchRouter{})
		setup.AddHandlers(NewStash(true))
	}
	return nil
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