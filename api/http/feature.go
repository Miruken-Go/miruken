package http

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/validate"
)

type (
	// Installer configure http client support.
	Installer struct {
		options Options
	}

	// ServerInstaller configures http server support
	ServerInstaller struct {
	}
)


// Installer

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{
		json.Feature(json.UseStandard()),
		validate.Feature(),
		&api.Installer{}}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.RegisterHandlers(&Router{})
		setup.AddBuilder(miruken.Options(i.options))
	}
	return nil
}

func WithOptions(options Options) func(installer *Installer) {
	return func(installer *Installer) {
		installer.options = options
	}
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


// ServerInstaller

func (i *ServerInstaller) DependsOn() []miruken.Feature {
	return []miruken.Feature{
		Feature(),
		&api.Installer{}}
}

func (i *ServerInstaller) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_serverFeatureTag) {
		setup.RegisterHandlers(&StatusCodeMapper{})
	}
	return nil
}

// ServerFeature configures http server support
func ServerFeature(
	config ... func(installer *ServerInstaller),
) miruken.Feature {
	installer := &ServerInstaller{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag, _serverFeatureTag byte