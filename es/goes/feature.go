package goes

import (
	"github.com/miruken-go/miruken/setup"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/event"
)

// Installer enables goes integration.
type Installer struct {
	cmdBus   command.Bus
	eventBus event.Bus
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&_featureTag) {

	}
	return nil
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

