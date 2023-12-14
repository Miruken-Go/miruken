package jwks

import (
	"github.com/miruken-go/miruken/setup"
)

// Installer enables core api support.
type Installer struct{}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		b.Specs(&KeySet{})
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
