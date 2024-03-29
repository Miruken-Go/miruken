package logs

import (
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken/setup"
)

// Installer configures logging support.
type Installer struct {
	root      logr.Logger
	verbosity int
}

func (i *Installer) SetVerbosity(verbosity int) {
	i.verbosity = verbosity
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&Factory{}).
			Handlers(&Factory{root: i.root}).
			Filters(&Emit{verbosity: i.verbosity})
	}
	return nil
}

// Verbosity sets the default Verbosity level when logging.
func Verbosity(verbosity int) func(*Installer) {
	return func(installer *Installer) {
		installer.SetVerbosity(verbosity)
	}
}

// Feature creates and configures logging support.
func Feature(
	rootLogger logr.Logger,
	config ...func(*Installer),
) setup.Feature {
	installer := &Installer{root: rootLogger}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
