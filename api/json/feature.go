package json

import (
	"github.com/miruken-go/miruken"
)

// Installer configure json support.
type Installer struct {
	mapper any
}

func (i *Installer) UseStandard() {
	i.mapper = &StdMapper{}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		mapper := i.mapper
		if miruken.IsNil(mapper) {
			mapper = &StdMapper{}
		}
		setup.Specs(mapper, &messageMapper{})
	}
	return nil
}

func UseStandard() func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseStandard()
	}
}

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