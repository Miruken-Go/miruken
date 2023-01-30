package swagger

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
)

// Installer configures http server support
type Installer struct {
	handlesPolicy miruken.Policy
}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{httpsrv.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		var handles miruken.Handles
		i.handlesPolicy = handles.Policy()
		setup.Observers(i)
	}
	return nil
}

func (i *Installer) BindingCreated(
	policy     miruken.Policy,
	descriptor *miruken.HandlerDescriptor,
	binding    miruken.Binding,
) {
	if policy == i.handlesPolicy {
		fmt.Println(binding)
	}
}

func (i *Installer) DescriptorCreated(
	_ *miruken.HandlerDescriptor,
) {
}

// Feature configures http server support
func Feature(
	config ...func(installer *Installer),
) miruken.Feature {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte
