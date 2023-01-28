package jsonstd

import (
	"github.com/miruken-go/miruken"
)

// Installer configure json support.
type Installer struct {}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&Mapper{}, &apiMessageMapper{})
	}
	return nil
}

func Feature(
	config ...func(installer *Installer),
) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte