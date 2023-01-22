package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/validate"
)

// Installer configure http client support.
type Installer struct {}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{
		json.Feature(json.UseStandard()),
		validate.Feature(),
		api.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.Specs(&Router{})
	}
	return nil
}

// Feature configures http client support
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