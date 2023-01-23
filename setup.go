package miruken

import (
	"container/list"
	"github.com/hashicorp/go-multierror"
)

type (
	// Feature encapsulates custom setup.
	Feature interface {
		Install(setup *SetupBuilder) error
	}
	FeatureFunc func(setup *SetupBuilder) error

	// SetupBuilder orchestrates the setup process.
	SetupBuilder struct {
		noInfer  bool
		handlers []any
		specs    []any
		features []Feature
		builders []Builder
		exclude  Predicate[HandlerSpec]
		factory  HandlerDescriptorFactory
		tags     map[any]Void
	}
)

func (f FeatureFunc) Install(setup *SetupBuilder) error {
	return f(setup)
}


// SetupBuilder

func (s *SetupBuilder) Features(
	features ... Feature,
) *SetupBuilder {
	s.features = append(s.features, features...)
	return s
}

func (s *SetupBuilder) Handlers(
	handlers ... any,
) *SetupBuilder {
	s.handlers = append(s.handlers, handlers...)
	return s
}

func (s *SetupBuilder) Specs(
	specs ... any,
) *SetupBuilder {
	s.specs = append(s.specs, specs...)
	return s
}

func (s *SetupBuilder) ExcludeSpecs(
	excludes ... Predicate[HandlerSpec],
) *SetupBuilder {
	s.exclude = CombinePredicates(s.exclude, excludes...)
	return s
}

func (s *SetupBuilder) Filters(
	providers ... FilterProvider,
) *SetupBuilder {
	return s.Builders(ProvideFilters(providers...))
}

func (s *SetupBuilder) Builders(
	builders ... Builder,
) *SetupBuilder {
	s.builders = append(s.builders, builders...)
	return s
}

func (s *SetupBuilder) Options(
	options ... any,
) *SetupBuilder {
	for _, option := range options {
		s.builders = append(s.builders, Options(option))
	}
	return s
}

func (s *SetupBuilder) UseHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) *SetupBuilder {
	s.factory = factory
	return s
}

func (s *SetupBuilder) WithoutInference() *SetupBuilder {
	s.noInfer = true
	return s
}

func (s *SetupBuilder) CanInstall(tag any) bool {
	if tags := s.tags; tags == nil {
		s.tags = map[any]Void{tag: {}}
		return true
	} else if _, found := tags[tag]; !found {
		tags[tag] = Void{}
		return true
	}
	return false
}

func (s *SetupBuilder) Handler() (handler Handler, buildErrors error) {
	factory := s.factory
	if IsNil(factory) {
		factory = NewMutableHandlerDescriptorFactory()
	}
	handler = &getHandlerDescriptorFactory{factory}

	buildErrors = s.installGraph(s.features)

	if specs := s.specs; len(specs) > 0 {
		hs := make([]HandlerSpec, 0, len(specs))
		exclude, noInfer := s.exclude, s.noInfer
		for _, spec := range specs {
			hspec := factory.MakeHandlerSpec(spec)
			if hspec == nil || (exclude != nil && exclude(hspec)) {
				continue
			}
			if noInfer {
				if _, _, err := factory.RegisterHandler(spec); err != nil {
					panic(err)
				}
			} else {
				hs = append(hs, hspec)
			}
		}

		if len(hs) > 0 {
			handler = &withHandler{handler, newInferenceHandler(factory, hs)}
		}
	}

	// Handler overrides
	if explicit := s.handlers; len(explicit) > 0 {
		handler = AddHandlers(handler, explicit...)
	}

	if builders := s.builders; len(builders) > 0 {
		handler = BuildUp(handler, builders...)
	}

	// call after setup hooks
	for _, feature := range s.features {
		if after, ok := feature.(interface{
			AfterInstall(*SetupBuilder, Handler) error
		}); ok {
			if err := after.AfterInstall(s, handler); err != nil {
				buildErrors = multierror.Append(buildErrors, err)
			}
		}
	}

	return handler, buildErrors
}

func (s *SetupBuilder) installGraph(
	features []Feature,
) (err error) {
	// traverse level-order so overrides can be applied in any order
	queue := list.New()
	for _, feature := range features {
		if !IsNil(feature) {
			queue.PushBack(feature)
		}
	}
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		feature := front.Value.(Feature)
		if dependsOn, ok := feature.(interface{
			DependsOn() []Feature
		}); ok {
			for _, dep := range dependsOn.DependsOn() {
				if !IsNil(dep) {
					queue.PushBack(dep)
				}
			}
		}
		if ie := feature.Install(s); ie != nil {
			err = multierror.Append(err, ie)
		}
	}
	return err
}

// GroupFeatures combines one or more Feature's into a single Feature.
func GroupFeatures(features ...Feature) FeatureFunc {
	return func(setup *SetupBuilder) error {
		for _, feature := range features {
			if !IsNil(feature) {
				if err := feature.Install(setup); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// Setup returns a new SetupBuilder with initial Feature's.
func Setup(features ...Feature) *SetupBuilder {
	return &SetupBuilder{features: features}
}
