package stdjson

import (
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/setup"
)

// Installer configure json support.
type Installer struct{}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{
		api.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&Mapper{}, &json.SurrogateMapper{}, &SurrogateMapper{})
	}
	return nil
}

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
