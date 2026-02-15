package es

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/es/aggregate"
	"github.com/miruken-go/miruken/es/command"
	"github.com/miruken-go/miruken/es/event"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal/seq"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
)

// Installer enables goes integration.
type Installer struct {
	provides miruken.Policy
	handles  miruken.Policy
	applies  miruken.Policy
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&_featureTag) {
		i.provides = (*provides.It)(nil).Policy()
		i.handles = (*handles.It)(nil).Policy()
		i.applies = (*event.Applies)(nil).Policy()
		b.Observers(i)
	}
	return nil
}

func (i *Installer) HandlerRuntimeBinding(
	runtime *miruken.HandlerRuntime,
	binding miruken.Binding,
	policy  miruken.Policy,
) {

}

func (i *Installer) HandlerRuntimeRegistered(
	runtime *miruken.HandlerRuntime,
) {
	model, err := i.makeAggregateModel(runtime)
	if err != nil {
		panic(err)
	} else if model != nil {
		fmt.Println("Aggregate", model)
	}
}

func (i *Installer) makeAggregateModel(
	runtime *miruken.HandlerRuntime,
) (m *aggregate.Model, err error) {
	ctor, ok := seq.First(
		seq.OfType[miruken.Binding, *miruken.CtorBinding](
			runtime.BindingsFor(i.provides)))
	if !ok {
		return nil, nil
	}

	aggMeta, ok := seq.First(
		seq.OfType[any, *aggregate.Metadata](
			slices.Values(ctor.Metadata())))
	if !ok {
		return nil, nil
	}

	aggType := ctor.LogicalOutputType()
	return aggregate.NewModel(aggType, aggMeta, nil)
}

func (i *Installer) makeCommandModel(
	binding miruken.Binding,
) (*command.Model, error) {
	cmdType, ok := binding.Key().(reflect.Type)
	if !ok {
		return nil, nil
	}

	cmdMeta, ok := seq.First(
		seq.OfType[any, *command.Metadata](
			slices.Values(binding.Metadata())))
	if !ok {
		return nil, nil
	}

	return command.NewModel(cmdType, cmdMeta)
}


// Feature discovers event sourcing components.
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
