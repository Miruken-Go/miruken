package json

import (
	"github.com/miruken-go/miruken"
)

// Installer configure json support.
type Installer struct {
	stdOptions      *StdOptions
	jsonIterOptions *IterOptions
	mapper           any
}

func (i *Installer) UseStandard(options ... StdOptions) {
	i.mapper = &StdMapper{}
}

func (i *Installer) UseJsonIterator(options ... IterOptions) {
	i.mapper = &IterMapper{}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		mapper := i.mapper
		if miruken.IsNil(mapper) {
			mapper = &StdMapper{}
		}
		setup.RegisterHandlers(mapper)
	}
	return nil
}

func UseStandard(options ... StdOptions) func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseStandard(options...)
	}
}

func UseJsonIterator(options ... IterOptions) func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseJsonIterator(options...)
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