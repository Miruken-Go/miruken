package miruken

type (
	// SetupBuilder configures a setup.
	SetupBuilder struct {
		noInfer  bool
		handlers []any
		specs    []any
		exclude  Predicate[HandlerSpec]
		factory  HandlerDescriptorFactory
		tags     map[any]struct{}
	}
	
	// Feature encapsulates custom setup.
	Feature interface {
		Install(setup *SetupBuilder)
	}
	FeatureFunc func(setup *SetupBuilder)
)

func (f FeatureFunc) Install(setup *SetupBuilder) { f(setup) }

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
	s.noInfer = false
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

func (s *SetupBuilder) Install(feature Feature) *SetupBuilder {
	feature.Install(s)
	return s
}

func (s *SetupBuilder) Build() Handler {
	factory := s.factory
	if IsNil(factory) {
		factory = NewMutableHandlerDescriptorFactory()
	}
	var handler Handler = &getHandlerDescriptorFactory{factory}

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

	return handler
}

var DisableInference = FeatureFunc(func(setup *SetupBuilder) {
	setup.noInfer = true
})

func WithHandlers(handlers ... any) Feature {
	return FeatureFunc(func(setup *SetupBuilder) {
		setup.AddHandlers(handlers...)
	})
}

func WithHandlerSpecs(specs ... any) Feature {
	return FeatureFunc(func(setup *SetupBuilder) {
		setup.RegisterHandlers(specs...)
	})
}

func ExcludeHandlerSpecs(rules ... Predicate[HandlerSpec]) Feature {
	return FeatureFunc(func(setup *SetupBuilder) {
		setup.Exclude(rules...)
	})
}

func WithoutInference(setup *SetupBuilder) {
	setup.DisableInference()
}

func WithHandlerDescriptorFactory(factory HandlerDescriptorFactory) Feature {
	return FeatureFunc(func(setup *SetupBuilder) {
		setup.SetHandlerDescriptorFactory(factory)
	})
}

func Setup(features ...Feature) Handler {
	setup := &SetupBuilder{}
	for _, feature := range features {
		if feature != nil {
			feature.Install(setup)
		}
	}
	return setup.Build()
}
