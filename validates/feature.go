package validates

import "github.com/miruken-go/miruken"

// Installer enables validation support.
type Installer struct {
	output bool
}

func (v *Installer) Output() {
	v.output = true
}

func (v *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&_featureTag) {
		setup.Filters(&Required{v.output})
	}
	return nil
}

func Output(installer *Installer) {
	installer.Output()
}

// Feature creates and configures validation support.
func Feature(config ...func(*Installer)) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var _featureTag byte
