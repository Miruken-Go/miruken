package setup

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
	return f.features
}

func (f *featureSet) Install(*Builder) error {
	return nil
}


// FeatureSet combines one or more Feature's into a single logical Feature.
func FeatureSet(features ...Feature) Feature {
	return &featureSet{features: features}
}
