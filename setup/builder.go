package setup

import (
	"container/list"
	"errors"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

// Builder orchestrates the setup process.
type Builder struct {
	noInfer   bool
	handlers  []any
	specs     []any
	features  []Feature
	builders  []miruken.Builder
	exclude   miruken.Predicate[miruken.HandlerSpec]
	factory   func([]miruken.BindingParser, []miruken.HandlerInfoObserver) miruken.HandlerInfoFactory
	parsers   []miruken.BindingParser
	observers []miruken.HandlerInfoObserver
	tags      map[any]struct{}
}

func (s *Builder) Features(
	features ...Feature,
) *Builder {
	s.features = append(s.features, features...)
	return s
}

func (s *Builder) Handlers(
	handlers ...any,
) *Builder {
	s.handlers = append(s.handlers, handlers...)
	return s
}

func (s *Builder) Specs(
	specs ...any,
) *Builder {
	s.specs = append(s.specs, specs...)
	return s
}

func (s *Builder) ExcludeSpecs(
	excludes ...miruken.Predicate[miruken.HandlerSpec],
) *Builder {
	s.exclude = miruken.CombinePredicates(s.exclude, excludes...)
	return s
}

func (s *Builder) Filters(
	providers ...miruken.FilterProvider,
) *Builder {
	return s.Builders(miruken.ProvideFilters(providers...))
}

func (s *Builder) Builders(
	builders ...miruken.Builder,
) *Builder {
	s.builders = append(s.builders, builders...)
	return s
}

func (s *Builder) With(
	values ...any,
) *Builder {
	s.builders = append(s.builders, miruken.With(values...))
	return s
}

func (s *Builder) Options(
	options ...any,
) *Builder {
	for _, option := range options {
		if builder, ok := option.(miruken.Builder); ok {
			s.builders = append(s.builders, builder)
		} else {
			s.builders = append(s.builders, miruken.Options(option))
		}
	}
	return s
}

func (s *Builder) Parsers(
	parsers ...miruken.BindingParser,
) *Builder {
	s.parsers = append(s.parsers, parsers...)
	return s
}

func (s *Builder) Observers(
	observers ...miruken.HandlerInfoObserver,
) *Builder {
	s.observers = append(s.observers, observers...)
	return s
}

func (s *Builder) Factory(
	factory func([]miruken.BindingParser, []miruken.HandlerInfoObserver) miruken.HandlerInfoFactory,
) *Builder {
	s.factory = factory
	return s
}

func (s *Builder) WithoutInference() *Builder {
	s.noInfer = true
	return s
}

func (s *Builder) Tag(tag any) bool {
	if tags := s.tags; tags == nil {
		s.tags = map[any]struct{}{tag: {}}
		return true
	} else if _, found := tags[tag]; !found {
		tags[tag] = struct{}{}
		return true
	}
	return false
}

func (s *Builder) Context() (*context.Context, error) {
	ctx, err := s.build()
	if err != nil {
		return nil, err
	}
	b, _, ok, err := miruken.Resolve[*bootstrapper](ctx)
	if ok {
		if _, err = b.bootstrap(ctx).Await(); err != nil {
			return nil, err
		}
	}
	return ctx, err
}

func (s *Builder) ContextAsync() *promise.Promise[*context.Context] {
	ctx, err := s.build()
	if err != nil {
		return promise.Reject[*context.Context](err)
	}
	b, _, ok, err := miruken.Resolve[*bootstrapper](ctx)
	if ok {
		return promise.Return(b.bootstrap(ctx), ctx)
	} else if err != nil {
		return promise.Reject[*context.Context](err)
	}
	return promise.Resolve(ctx)
}

func (s *Builder) build() (*context.Context, error) {
	buildErrors := s.installGraph(s.features)

	var factory miruken.HandlerInfoFactory
	if f := s.factory; f != nil {
		factory = f(s.parsers, s.observers)
	}
	if factory == nil {
		var builder miruken.HandlerInfoFactoryBuilder
		factory = builder.
			Parsers(s.parsers...).
			Observers(s.observers...).
			Build()
	}

	var handler miruken.Handler = &miruken.CurrentHandlerInfoFactoryProvider{Factory: factory}

	specs := append(s.specs, &bootstrapper{})
	hs := make([]miruken.HandlerSpec, 0, len(specs))
	exclude, noInfer := s.exclude, s.noInfer
	for _, spec := range specs {
		h := factory.Spec(spec)
		if h == nil || (exclude != nil && exclude(h)) {
			continue
		}
		if noInfer {
			if _, _, err := factory.Register(spec); err != nil {
				panic(err)
			}
		} else {
			hs = append(hs, h)
		}
	}

	if len(hs) > 0 {
		handler = miruken.AddHandlers(handler, miruken.NewInferenceHandler(factory, hs))
	}

	// Context overrides
	if explicit := s.handlers; len(explicit) > 0 {
		handler = miruken.AddHandlers(handler, explicit...)
	}

	if builders := s.builders; len(builders) > 0 {
		handler = miruken.BuildUp(handler, builders...)
	}

	ctx := context.New(handler)

	// call after setup hooks
	for _, feature := range s.features {
		if after, ok := feature.(interface {
			AfterInstall(*Builder, miruken.Handler) error
		}); ok {
			if err := after.AfterInstall(s, ctx); err != nil {
				buildErrors = errors.Join(buildErrors, err)
			}
		}
	}

	return ctx, buildErrors
}

func (s *Builder) installGraph(
	features []Feature,
) (err error) {
	// traverse level-order so overrides can be applied in any order
	queue := list.New()
	for _, feature := range features {
		if !internal.IsNil(feature) {
			queue.PushBack(feature)
		}
	}
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		feature := front.Value.(Feature)
		if dependsOn, ok := feature.(interface {
			DependsOn() []Feature
		}); ok {
			for _, dep := range dependsOn.DependsOn() {
				if !internal.IsNil(dep) {
					queue.PushBack(dep)
				}
			}
		}
		if ie := feature.Install(s); ie != nil {
			err = errors.Join(err, ie)
		}
	}
	return err
}

// New returns a new Builder with initial Feature's.
func New(features ...Feature) *Builder {
	return &Builder{features: features}
}
