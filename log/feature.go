package log

import (
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
)

// Installer configures logging support.
type Installer struct {
	root      logr.Logger
	verbosity int
}

func (v *Installer) SetVerbosity (verbosity int) {
	v.verbosity = verbosity
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		setup.RegisterHandlers(&factory{})
		setup.AddHandlers(&factory{v.root})
		setup.RequireFilters(NewProvider(v.verbosity))
	}
	return nil
}

// Verbosity sets the default verbosity level when logging.
func Verbosity(verbosity int) func(installer *Installer) {
	return func(installer *Installer) {
		installer.SetVerbosity(verbosity)
	}
}

func Feature(
	rootLogger logr.Logger,
	config ... func(installer *Installer),
) miruken.Feature {
	installer := &Installer{root: rootLogger}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag byte

