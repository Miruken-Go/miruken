package http

import (
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/setup"
	"github.com/miruken-go/miruken/validates"
)

// Installer configure http client support.
type Installer struct{}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{
		validates.Feature(),
		api.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&Router{})
	}
	return nil
}

// Feature configures http client support
func Feature(config ...func(*Installer)) setup.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
