package setup

import "github.com/miruken-go/miruken/internal"

type (
	// Feature encapsulates custom setup.
	Feature interface {
		Install(*Builder) error
	}
	FeatureFunc func(*Builder) error
)

func (f FeatureFunc) Install(setup *Builder) error {
	return f(setup)
}


// FeatureSet combines one or more Feature's into a single logical Feature.
func FeatureSet(features ...Feature) FeatureFunc {
	return func(setup *Builder) error {
		for _, feature := range features {
			if !internal.IsNil(feature) {
				if err := feature.Install(setup); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
