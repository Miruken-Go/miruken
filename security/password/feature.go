package password

import "github.com/miruken-go/miruken"

// Installer enables user/password authentication.
type Installer struct {}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		setup.Specs(&LoginModule{})
	}
	return nil
}

func Feature(config ...func(*Installer)) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte

