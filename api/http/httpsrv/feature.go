package httpsrv

import (
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/setup"
)

// Installer configures http server support
type Installer struct {}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{http.Feature()}
}

func (i *Installer) Install(setup *setup.Builder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(
			&PolyHandler{},
			&StatusCodeMapper{})
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