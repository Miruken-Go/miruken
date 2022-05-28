package miruken

import "github.com/hashicorp/go-multierror"

type (
	// Feature encapsulates custom setup.
	Feature interface {
		Install(setup *SetupBuilder) error
	}
	InstallFeature func(setup *SetupBuilder) error

	// SetupBuilder orchestrates the setup process.
	SetupBuilder struct {
		noInfer  bool
		handlers []any
		specs    []any
		features []Feature
		exclude  Predicate[HandlerSpec]
		factory  HandlerDescriptorFactory
		tags     map[any]struct{}
	}
)

func (f InstallFeature) Install(
	setup *SetupBuilder,
) error {
	return f(setup)
}

func (s *SetupBuilder) AddHandlers(
	handlers ... any,
) *SetupBuilder {
	s.handlers = append(s.handlers, handlers...)
	return s
}

func (s *SetupBuilder) RegisterHandlers(
	specs ... any,
) *SetupBuilder {
	s.specs = append(s.specs, specs...)
	return s
}

func (s *SetupBuilder) Exclude(
	excludes ... Predicate[HandlerSpec],
) *SetupBuilder {
	s.exclude = CombinePredicates(s.exclude, excludes...)
	return s
}

func (s *SetupBuilder) AddFilters(
	providers ... FilterProvider,
) *SetupBuilder {
	var handles Handles
	handles.Policy().AddFilters(providers...)
	return s
}

func (s *SetupBuilder) SetHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) *SetupBuilder {
	s.factory = factory
	return s
}

func (s *SetupBuilder) DisableInference() {
	s.noInfer = true
}

func (s *SetupBuilder) CanInstall(tag any) bool {
	if tags := s.tags; tags == nil {
		s.tags = map[any]struct{} { tag: {} }
		return true
	} else if _, found := tags[tag]; !found {
		tags[tag] = struct{}{}
		return true
	}
	return false
}

func (s *SetupBuilder) Install(features ...Feature) *SetupBuilder {
	s.features = append(s.features, features...)
	return s
}

func (s *SetupBuilder) Build() (handler Handler, buildErrors error) {
	factory := s.factory
	if IsNil(factory) {
		factory = NewMutableHandlerDescriptorFactory()
	}
	handler = &getHandlerDescriptorFactory{factory}

	// Use index and not range to allow features to install other features
	for i := 0; i < len(s.features); i++ {
		if feature := s.features[i]; feature != nil {
			if err := feature.Install(s); err != nil {
				buildErrors = multierror.Append(buildErrors, err)
			}
		}
	}

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

func Handlers(handlers ... any) InstallFeature {
	return func(setup *SetupBuilder) error {
		setup.AddHandlers(handlers...)
		return nil
	}
}

func HandlerSpecs(specs ... any) InstallFeature {
	return func(setup *SetupBuilder) error {
		setup.RegisterHandlers(specs...)
		return nil
	}
}

func ExcludeHandlerSpecs(rules ... Predicate[HandlerSpec]) InstallFeature {
	return func(setup *SetupBuilder) error {
		setup.Exclude(rules...)
		return nil
	}
}

var NoInference InstallFeature = func(setup *SetupBuilder) error {
	setup.DisableInference()
	return nil
}

func UseHandlerDescriptorFactory(factory HandlerDescriptorFactory) InstallFeature {
	return func(setup *SetupBuilder) error {
		setup.SetHandlerDescriptorFactory(factory)
		return nil
	}
}

func Setup(features ...Feature) (Handler, error) {
	setup := &SetupBuilder{}
	return setup.Install(features...).Build()
}
