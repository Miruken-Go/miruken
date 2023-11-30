package api

import (
	"github.com/miruken-go/miruken"
)

// Installer enables core api support.
type Installer struct {}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(
			&Stash{},
			&Scheduler{},
			&PassThroughRouter{},
			&batchRouter{},
			&MultipartMapper{}).
			Handlers(NewStash(true))
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