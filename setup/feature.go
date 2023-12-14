package setup

import "github.com/miruken-go/miruken/internal"

type (
	// Feature encapsulates custom setup.
	Feature interface {
		Install(*Builder) error
	}
	FeatureFunc func(*Builder) error
)

func (f FeatureFunc) Install(b *Builder) error {
	return f(b)
}

// FeatureSet combines one or more Feature's into a single logical Feature.
func FeatureSet(features ...Feature) FeatureFunc {
	return func(b *Builder) error {
		for _, feature := range features {
			if !internal.IsNil(feature) {
				if err := feature.Install(b); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
