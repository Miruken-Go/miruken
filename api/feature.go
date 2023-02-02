package api

import "github.com/miruken-go/miruken"

// Installer enables api support.
type Installer struct {}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(
			&Stash{},
			&Scheduler{},
			&PassThroughRouter{},
			&batchRouter{}).
			Handlers(NewStash(true))
	}
	return nil
}

func Feature(config ...func(installer *Installer)) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte