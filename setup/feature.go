package setup

import (
	"github.com/miruken-go/miruken/internal"
)

type (
	// Feature encapsulates custom setup.
	Feature interface {
		Install(*Builder) error
	}
	FeatureFunc func(*Builder) error

	featureSet struct {
		features []Feature
	}
)

func (f FeatureFunc) Install(b *Builder) error {
	return f(b)
}


// featureSet

func (f *featureSet) DependsOn() []Feature {
	var deps []Feature
	for _, feature := range f.features {
		if !internal.IsNil(feature) {
			if dependsOn, ok := feature.(interface {
				DependsOn() []Feature
			}); ok {
				deps = append(deps, dependsOn.DependsOn()...)
			}
		}
	}
	return deps
}

func (f *featureSet) Install(b *Builder) error {
	if b.Tag(f) {
		for _, feature := range f.features {
			if !internal.IsNil(feature) {
				if err := feature.Install(b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}


// FeatureSet combines one or more Feature's into a single logical Feature.
func FeatureSet(features ...Feature) Feature {
	return &featureSet{features: features}
}
