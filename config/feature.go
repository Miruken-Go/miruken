package config

import (
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/setup"
)

type (
	// Provider defines the api to allow configuration
	// providers to expose their configuration information.
	Provider interface {
		Unmarshal(path string, flat bool, output any) error
	}

	// Installer enables configuration support.
	Installer struct {
		provider Provider
	}
)

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		if provider := i.provider; !internal.IsNil(provider) {
			b.Specs(&Factory{}).
				Handlers(&Factory{Provider: i.provider})
		}
	}
	return nil
}

// Feature creates and configures configuration support
// using the supplied configuration Provider.
func Feature(
	provider Provider,
	config ...func(*Installer),
) setup.Feature {
	if internal.IsNil(provider) {
		panic("provider cannot be nil")
	}
	installer := &Installer{provider}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
