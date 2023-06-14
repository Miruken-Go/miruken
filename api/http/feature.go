package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/validates"
)

// Installer configure http client support.
type Installer struct {}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{
		validates.Feature(),
		api.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&Router{})
	}
	return nil
}

// Feature configures http client support
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