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
	if setup.Tag(&featureTag) {
		setup.Specs(&Factory{}).
			  Handlers(&Factory{v.root}).
			  Filters(&Provider{v.verbosity})
	}
	return nil
}

// Verbosity sets the default Verbosity level when logging.
func Verbosity(verbosity int) func(installer *Installer) {
	return func(installer *Installer) {
		installer.SetVerbosity(verbosity)
	}
}

// Feature creates and configures logging support.
func Feature(
	rootLogger logr.Logger,
	config     ...func(installer *Installer),
) miruken.Feature {
	installer := &Installer{root: rootLogger}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte

