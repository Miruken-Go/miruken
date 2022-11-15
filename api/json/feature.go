package json

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/miruken-go/miruken"
)

// Installer configure json support.
type Installer struct {
	options any
	mapper  any
}

func (i *Installer) UseStandard(options *StdOptions) {
	i.mapper = &StdMapper{}
	if !miruken.IsNil(options) {
		i.options = *options
	}
}

func (i *Installer) UseJsonIterator(options *IterOptions) {
	i.mapper  = &IterMapper{}
	if !miruken.IsNil(options) {
		i.options = *options
	}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.CanInstall(&_featureTag) {
		mapper := i.mapper
		if miruken.IsNil(mapper) {
			mapper = &StdMapper{}
		}
		setup.RegisterHandlers(mapper)
		if options := i.options; !miruken.IsNil(options) {
			setup.AddBuilder(miruken.Options(options))
		}
	}
	return nil
}

func UseStandard() func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseStandard(nil)
	}
}

func UseStandardWithOptions(options StdOptions) func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseStandard(&options)
	}
}

func UseJsonIterator() func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseJsonIterator(nil)
	}
}

func UseJsonIteratorWithConfig(config jsoniter.Config) func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseJsonIterator(&IterOptions{Config: config})
	}
}

func UseJsonIteratorInstance(instance jsoniter.API) func(installer *Installer) {
	return func(installer *Installer) {
		installer.UseJsonIterator(&IterOptions{Api: instance})
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