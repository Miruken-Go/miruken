package validates

import (
	"github.com/miruken-go/miruken/setup"
)

// Installer enables validation support.
type Installer struct {
	output bool
}

func (i *Installer) Output() {
	i.output = true
}

func (i *Installer) Install(setup *setup.Builder) error {
	if setup.Tag(&_featureTag) {
		setup.Filters(&Required{i.output})
	}
	return nil
}

func Output(installer *Installer) {
	installer.Output()
}

// Feature creates and configures validation support.
func Feature(config ...func(*Installer)) setup.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag byte
