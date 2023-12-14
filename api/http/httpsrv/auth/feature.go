package auth

import (
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/setup"
)

// Installer configures http server support
type Installer struct{}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{httpsrv.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs()
	}
	return nil
}

// Feature configures http server support
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
