package es

import (
	"slices"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal/seq"
	"github.com/miruken-go/miruken/setup"
)

// Installer enables goes integration.
type Installer struct {
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&_featureTag) {
		b.Observers(i)
	}
	return nil
}

func (i *Installer) HandlerRuntimeRegistered(
	runtime *miruken.HandlerRuntime,
) {
	for _, bindings := range runtime.Bindings() {
		for binding := range bindings {
			for model := range seq.OfType[any, interface{
				InitWithBinding(miruken.Binding) error
			}](slices.Values(binding.Metadata())) {
				if err := model.InitWithBinding(binding); err != nil {
					panic(err)
				}
			}
		}
	}
}

// Feature creates and configures goes integration.
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

