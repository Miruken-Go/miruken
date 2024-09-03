package es

import (
	"fmt"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/es/aggregate"
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
	spec, ok := runtime.Spec().(*miruken.TypeSpec)
	if !ok {
		return
	}

	for _, bindings := range runtime.Bindings() {
		for binding := range bindings {
			fmt.Println(binding)
		}
	}

	_, err := aggregate.NewModel(spec.Type(), "", nil)
	if err != nil {
		panic(err)
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

