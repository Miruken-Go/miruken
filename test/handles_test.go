package test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	Counter interface {
		Count() int
		Inc() int
	}

	Counted struct {
		count int
	}

	Foo struct{ Counted }
	Bar struct{ Counted }
	Baz struct{ Counted }
	Bam struct{ Counted }
	Boo struct{ Counted }
)

func (c *Counted) Count() int {
	return c.count
}

func (c *Counted) Inc() int {
	c.count++
	return c.count
}

// FooHandler
type FooHandler struct{}

func (h *FooHandler) Handle(
	callback any,
	greedy bool,
	composer miruken.Handler,
) miruken.HandleResult {
	switch foo := callback.(type) {
	case *Foo:
		foo.Inc()
		return composer.Handle(Bar{}, false, nil)
	default:
		return miruken.NotHandled
	}
}

// BarHandler
type BarHandler struct{}

func (h *BarHandler) HandleBar(
	_ *handles.It, _ Bar,
) {
}

// CounterHandler
type CounterHandler struct{}

func (h *CounterHandler) HandleCounted(
	_ *handles.It, counter Counter,
) (Counter, miruken.HandleResult) {
	switch c := counter.Inc(); {
	case c > 0 && c%3 == 0:
		err := fmt.Errorf("%v is divisible by 3", c)
		return nil, miruken.NotHandled.WithError(err)
	case c%2 == 0:
		return nil, miruken.NotHandled
	default:
		return counter, miruken.Handled
	}
}

// CountByOneHandler
type CountByTwoHandler struct{}

func (h *CountByTwoHandler) HandleCounted(
	_ *handles.It, counter Counter,
) (Counter, miruken.HandleResult) {
	counter.Inc()
	counter.Inc()
	return counter, miruken.Handled
}

// MultiHandler
type MultiHandler struct {
	foo Foo
	bar Bar
}

func (h *MultiHandler) HandleFoo(
	_ *handles.It, foo *Foo,
	composer miruken.Handler,
) error {
	h.foo.Inc()
	if foo.Inc() == 5 {
		return errors.New("count reached 5")
	}
	composer.Handle(new(Bar), false, nil)
	return nil
}

func (h *MultiHandler) HandleBar(
	_ *handles.It, bar *Bar,
) miruken.HandleResult {
	h.bar.Inc()
	if bar.Inc()%2 == 0 {
		return miruken.Handled
	}
	return miruken.NotHandled
}

// EverythingHandler
type EverythingHandler struct{}

func (h *EverythingHandler) HandleEverything(
	_ *handles.It, callback any,
) miruken.HandleResult {
	switch cb := callback.(type) {
	case *Foo:
		cb.Inc()
		return miruken.Handled
	case Counter:
		cb.Inc()
		cb.Inc()
		return miruken.Handled
	default:
		return miruken.NotHandled
	}
}

// EverythingImplicitHandler
type EverythingImplicitHandler struct{}

func (h *EverythingImplicitHandler) HandleEverything(
	it *handles.It,
) miruken.HandleResult {
	switch cb := it.Source().(type) {
	case *Bar:
		cb.Inc()
		cb.Inc()
		return miruken.Handled
	case Counter:
		cb.Inc()
		cb.Inc()
		cb.Inc()
		return miruken.Handled
	default:
		return miruken.NotHandled
	}
}

// EverythingSpecHandler
type EverythingSpecHandler struct{}

func (h *EverythingSpecHandler) HandleEverything(
	_ *struct{ handles.It }, callback any,
) miruken.HandleResult {
	switch cb := callback.(type) {
	case *Baz:
		cb.Inc()
		return miruken.Handled
	case Counter:
		cb.Inc()
		cb.Inc()
		return miruken.Handled
	default:
		return miruken.NotHandled
	}
}

// SpecificationHandler
type SpecificationHandler struct{}

func (h *SpecificationHandler) HandleFoo(
	_ *struct {
		handles.It
		handles.Strict
	}, foo *Foo,
) miruken.HandleResult {
	foo.Inc()
	return miruken.Handled
}

// DependencyHandler
type DependencyHandler struct{}

func (h *DependencyHandler) RequiredDependency(
	_ *handles.It, foo *Foo,
	bar *Bar,
) {
	if bar == nil {
		panic("bar cannot be nil")
	}
	foo.Inc()
}

func (h *DependencyHandler) RequiredSliceDependency(
	_ *handles.It, boo *Boo,
	bars []*Bar,
) {
	boo.Inc()
	for _, bar := range bars {
		bar.Inc()
	}
}

func (h *DependencyHandler) OptionalDependency(
	_ *handles.It, bar *Bar,
	_ *struct{ args.Optional }, foo *Foo,
) {
	bar.Inc()
	if foo != nil {
		foo.Inc()
	}
}

func (h *DependencyHandler) OptionalSliceDependency(
	_ *handles.It, baz *Baz,
	_ *struct{ args.Optional }, bars []*Bar,
) {
	baz.Inc()
	for _, bar := range bars {
		bar.Inc()
	}
}

func (h *DependencyHandler) StrictDependency(
	_ *handles.It, bam *Bam,
	_ *struct{ args.Strict }, bars []*Bar,
) {
	bam.Inc()
	for _, bar := range bars {
		bar.Inc()
	}
}

type (
	DateFormat string

	Config struct {
		baseUrl string
		timeout int
		created string
	}

	Configuration struct {
		config *Config
	}

	DefaultConfiguration struct {
		miruken.BindingGroup
		Configuration
		DateFormat `layout:"02 Jan 06 15:04 MST"`
	}
)

func (f *DateFormat) InitWithTag(tag reflect.StructTag) error {
	if layout, ok := tag.Lookup("layout"); ok {
		*f = DateFormat(layout)
	}
	return nil
}

func (c *Configuration) Validate(
	typ reflect.Type,
	_ miruken.DependencyArg,
) error {
	if !reflect.TypeOf(c.config).AssignableTo(typ) {
		return fmt.Errorf("the Configuration resolver expects a %T field", c.config)
	}
	return nil
}

func (c *Configuration) Resolve(
	typ reflect.Type,
	dep miruken.DependencyArg,
	ctx miruken.HandleContext,
) (reflect.Value, *promise.Promise[reflect.Value], error) {
	if c.config == nil {
		c.config = &Config{
			baseUrl: "https://server/api",
			timeout: 30000,
		}
		var layout string
		if format, ok := slices.First(slices.OfType[any, DateFormat](dep.Metadata())); ok {
			layout = string(format)
		} else {
			layout = "Mon, 02 Jan 2006 15:04:05 MST"
		}
		c.config.created = time.Now().Format(layout)
	}
	return reflect.ValueOf(c.config), nil, nil
}

// DependencyResolverHandler
type DependencyResolverHandler struct{}

func (h *DependencyResolverHandler) UseDependencyResolver(
	_ *handles.It, foo *Foo,
	_ *struct{ DefaultConfiguration }, config *Config,
) *Config {
	foo.Inc()
	return config
}

// MixedHandler
type MixedHandler struct{}

func (m *MixedHandler) Mix(
	_ *struct {
		h handles.It
		m maps.It
	}, callback miruken.Callback,
) string {
	switch cb := callback.(type) {
	case *handles.It:
		return fmt.Sprintf("Handles %T", cb.Source())
	case *maps.It:
		return fmt.Sprintf("It %T", cb.Source())
	default:
		return ""
	}
}

// SimpleAsyncHandler
type SimpleAsyncHandler struct{}

func (h *SimpleAsyncHandler) HandleBar(
	_ *handles.It, bar *Bar,
) *promise.Promise[*Bar] {
	bar.Inc()
	return promise.Then(
		promise.Delay[any](nil, time.Duration(bar.Count())*time.Millisecond),
		func(any) *Bar { return bar })
}

func (h *SimpleAsyncHandler) HandleBoo(
	_ *handles.It, boo *Boo,
	baz *Baz,
) *promise.Promise[*Baz] {
	boo.Inc()
	baz.Inc()
	return promise.Resolve(baz)
}

func (h *SimpleAsyncHandler) HandleBamPromiseArg(
	_ *handles.It, bam *Bam,
	baz *promise.Promise[*Baz],
) *Baz {
	bam.Inc()
	bam.Inc()
	buz, _ := baz.Await()
	buz.Inc()
	return buz
}

func (h *SimpleAsyncHandler) HandleFooPromiseArgLift(
	_ *handles.It, foo *Foo,
	boo *promise.Promise[*Boo],
) *Boo {
	foo.Inc()
	foo.Inc()
	boz, _ := boo.Await()
	boz.Inc()
	return boz
}

func (h *SimpleAsyncHandler) ProvidesBaz(
	_ *provides.It,
) *promise.Promise[*Baz] {
	return promise.Resolve(new(Baz))
}

func (h *SimpleAsyncHandler) ProvidesBoo(
	_ *provides.It,
) *Boo {
	return &Boo{Counted{5}}
}

// ComplexAsyncHandler
type ComplexAsyncHandler struct{}

func (h *ComplexAsyncHandler) HandleFoo(
	_ *struct {
		handles.It
		handles.Strict
	}, foo *Foo,
	baz []*Baz,
) []*Baz {
	foo.Inc()
	return baz
}

func (h *ComplexAsyncHandler) ProvidesBaz(
	_ *provides.It,
) *Baz {
	return new(Baz)
}

func (h *ComplexAsyncHandler) ProvidesBazAsync(
	_ *provides.It,
) *promise.Promise[*Baz] {
	return promise.Resolve(new(Baz))
}

// ErrorAsyncHandler
type ErrorAsyncHandler struct{}

func (h *ErrorAsyncHandler) HandleFoo(
	_ *handles.It, foo *Foo,
) *promise.Promise[*Bar] {
	return promise.Reject[*Bar](fmt.Errorf("bad Foo %p", foo))
}

// InvalidHandler
type InvalidHandler struct{}

func (h *InvalidHandler) Constructor() {}

func (h *InvalidHandler) NoConstructor() {}

func (h *InvalidHandler) MissingDependency(
	_ *handles.It, _ *Bar,
	_ *struct{},
) {
}

/* Relaxed for implicit cascades */
func (h *InvalidHandler) TooManyReturnValues(
	_ *handles.It, _ *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (h *InvalidHandler) SecondReturnMustBeErrorOrHandleResult(
	_ *handles.It, _ *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}
/**/

func (h *InvalidHandler) UntypedInterfaceDependency(
	_ *handles.It, _ *Bar,
	any any,
) miruken.HandleResult {
	return miruken.Handled
}

func (h *InvalidHandler) CallbackInterfaceSpec(
	*struct{ miruken.Callback },
) miruken.HandleResult {
	return miruken.Handled
}

func (h *InvalidHandler) MissingCallbackArgument(
	*struct{ handles.It },
) miruken.HandleResult {
	return miruken.Handled
}

// Anonymous metadata
type Anonymous struct{}

type TransactionalMode byte

const (
	TransactionalSupports = TransactionalMode(iota)
	TransactionalRequired
	TransactionalRequiresNew
)

// Transactional metadata
type Transactional struct {
	mode TransactionalMode
}

func (t *Transactional) Init() error {
	t.mode = TransactionalRequired
	return nil
}

func (t *Transactional) InitWithTag(tag reflect.StructTag) error {
	if mode, ok := tag.Lookup("mode"); ok {
		switch mode {
		case "supports":
			t.mode = TransactionalSupports
		case "required":
			t.mode = TransactionalRequired
		case "requiresNew":
			t.mode = TransactionalRequiresNew
		default:
			return fmt.Errorf("unrecognized transactional mode %q", mode)
		}
	}
	return nil
}

// MetadataHandler
type MetadataHandler struct{}

func (m *MetadataHandler) HandleFoo(
	_ *struct {
		handles.It
		Transactional `mode:"requiresNew"`
	}, foo *Foo,
	ctx miruken.HandleContext,
) Transactional {
	foo.Inc()
	if transactional, ok :=
		slices.First(slices.OfType[any, Transactional](
			ctx.Binding.Metadata())); ok {
		return transactional
	}
	return Transactional{}
}

func (m *MetadataHandler) HandleBar(
	_ *struct {
		handles.It
		handles.Strict
		Anonymous
	}, bar *Bar,
	ctx miruken.HandleContext,
) []Anonymous {
	bar.Inc()
	bar.Inc()
	return slices.OfType[any, Anonymous](ctx.Binding.Metadata())
}

// MetadataInvalidHandler
type MetadataInvalidHandler struct{}

func (m *MetadataInvalidHandler) HandleFoo(
	_ *struct {
		handles.It
		Transactional `mode:"suppress"`
	}) {
}

func HandleFoo(
	_ *handles.It, foo *Foo,
) miruken.HandleResult {
	foo.Inc()
	return miruken.Handled
}

func HandleCounted(
	_ *struct{ handles.It }, counter Counter,
) {
	counter.Inc()
	counter.Inc()
}

type HandlesTestSuite struct {
	suite.Suite
}

func (suite *HandlesTestSuite) Setup() (miruken.Handler, error) {
	return setup.New(TestFeature).ExcludeSpecs(
		func(spec miruken.HandlerSpec) bool {
			switch ts := spec.(type) {
			case miruken.TypeSpec:
				return strings.Contains(ts.Name(), "Invalid")
			default:
				return false
			}
		}).Context()
}

func (suite *HandlesTestSuite) TestHandles() {
	suite.Run("Invariant", func() {
		handler, _ := setup.New().
			Specs(&FooHandler{}, &BarHandler{}).
			Handlers(new(FooHandler), new(BarHandler)).
			Context()
		foo := new(Foo)
		result := handler.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Contravariant", func() {
		handler, _ := setup.New().
			Specs(&CounterHandler{}).
			Handlers(new(CounterHandler)).
			Context()
		foo := new(Foo)
		result := handler.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("PointerIndirect", func() {
		handler, _ := setup.New().
			Specs(&BarHandler{}).
			Handlers(new(BarHandler)).
			Context()
		var bar Bar
		result := handler.Handle(bar, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		result = handler.Handle(&bar, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
	})

	suite.Run("HandleResult", func() {
		handler, _ := setup.New().
			Specs(&CounterHandler{}).
			Context()
		suite.Run("Handled", func() {
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
		})

		suite.Run("NotHandled", func() {
			foo := new(Foo)
			foo.Inc()
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
		})

		suite.Run("NotHandled With Error", func() {
			foo := new(Foo)
			foo.Inc()
			foo.Inc()
			result := handler.Handle(foo, false, nil)
			suite.True(result.IsError())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			suite.Equal("3 is divisible by 3", result.Error().Error())
		})
	})

	suite.Run("Multiple", func() {
		multi := new(MultiHandler)
		handler, _ := setup.New().
			Specs(&MultiHandler{}).
			Handlers(multi).
			Context()

		foo := new(Foo)
		for i := range 4 {
			result := handler.Handle(foo, false, nil)
			suite.Equal(miruken.Handled, result)
			suite.Equal(i+1, foo.Count())
		}

		suite.Equal(4, multi.foo.Count())
		suite.Equal(4, multi.bar.Count())

		result := handler.Handle(foo, false, nil)
		suite.True(result.IsError())
		suite.Equal("count reached 5", result.Error().Error())

		suite.Equal(5, multi.foo.Count())
		suite.Equal(4, multi.bar.Count())
	})

	suite.Run("Everything", func() {
		handler, _ := setup.New().
			Specs(&EverythingHandler{}).
			Handlers(new(EverythingHandler)).
			Context()

		suite.Run("Invariant", func() {
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)

			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())

			result = handler.Handle(foo, false, nil)

			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, foo.Count())
		})

		suite.Run("Contravariant", func() {
			bar := new(Bar)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Count())
		})
	})

	suite.Run("EverythingImplicit", func() {
		handler, _ := setup.New().
			Specs(&EverythingImplicitHandler{}).
			Handlers(new(EverythingImplicitHandler)).
			Context()

		suite.Run("Invariant", func() {
			bar := new(Bar)
			result := handler.Handle(bar, false, nil)

			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Count())
		})

		suite.Run("Contravariant", func() {
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, foo.Count())
		})
	})

	suite.Run("EverythingSpec", func() {
		handler, _ := setup.New().
			Specs(&EverythingSpecHandler{}).
			Handlers(new(EverythingSpecHandler)).
			Context()

		suite.Run("Invariant", func() {
			baz := new(Baz)
			result := handler.Handle(baz, false, nil)

			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, baz.Count())
		})

		suite.Run("Contravariant", func() {
			bar := new(Bar)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Count())
		})
	})

	suite.Run("Specification", func() {
		handler, _ := setup.New().
			Specs(&SpecificationHandler{}).
			Handlers(new(SpecificationHandler)).
			Context()
		suite.Run("Strict", func() {
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})
	})

	suite.Run("Dependencies", func() {
		handler, _ := setup.New().
			Specs(&DependencyHandler{}).
			Handlers(new(DependencyHandler)).
			Context()
		suite.Run("Required", func() {
			defer func() {
				if r := recover(); r != nil {
					var err *miruken.MethodBindingError
					if errors.As(r.(error), &err) {
						suite.Equal("RequiredDependency", err.Method.Name)
					} else {
						suite.Fail("Expected MethodBindingError")
					}
				}
			}()
			handler.Handle(new(Foo), false, nil)
		})

		suite.Run("RequiredSlice", func() {
			boo := new(Boo)
			bars := []any{new(Bar), new(Bar)}
			result := miruken.BuildUp(handler, provides.With(bars...)).Handle(boo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, boo.Count())
			for _, bar := range bars {
				suite.Equal(1, bar.(*Bar).Count())
			}
		})

		suite.Run("Optional", func() {
			bar := new(Bar)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bar.Count())
		})

		suite.Run("OptionalWithValue", func() {
			bar := new(Bar)
			foo := new(Foo)
			result := miruken.BuildUp(handler, provides.With(foo)).Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bar.Count())
			suite.Equal(1, foo.Count())
		})

		suite.Run("OptionalSlice", func() {
			baz := new(Baz)
			bars := []any{new(Bar), new(Bar)}
			result := handler.Handle(baz, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, baz.Count())
			result = miruken.BuildUp(handler, provides.With(bars...)).Handle(baz, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, baz.Count())
			for _, bar := range bars {
				suite.Equal(1, bar.(*Bar).Count())
			}
		})

		suite.Run("StrictSlice", func() {
			bam := new(Bam)
			bars1 := []any{new(Bar), new(Bar)}
			result := miruken.BuildUp(handler, provides.With(bars1...)).Handle(bam, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
			bars2 := []*Bar{new(Bar), new(Bar)}
			result = miruken.BuildUp(handler, provides.With(bars2)).Handle(bam, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bam.Count())
			for _, bar := range bars2 {
				suite.Equal(1, bar.Count())
			}
		})

		suite.Run("CustomResolver", func() {
			handler, _ := setup.New().
				Specs(&DependencyResolverHandler{}).
				Handlers(new(DependencyResolverHandler)).
				Context()
			if config, _, err := miruken.Execute[*Config](handler, new(Foo)); err == nil {
				suite.NotNil(*config)
				suite.Equal("https://server/api", config.baseUrl)
				suite.Equal(30000, config.timeout)
				_, err := time.Parse(time.RFC822, config.created)
				suite.Nil(err)
				_, err = time.Parse(time.RFC3339, config.created)
				suite.IsType(&time.ParseError{}, err)
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})
	})

	suite.Run("Metadata", func() {
		suite.Run("Simple", func() {
			handler, _ := setup.New().
				Specs(&MetadataHandler{}).
				Context()
			bar := new(Bar)
			if anonymous, _, err := miruken.Execute[[]Anonymous](handler, bar); err == nil {
				suite.Len(anonymous, 1)
				suite.Equal(2, bar.Count())
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Pointer", func() {
			handler, _ := setup.New().
				Specs(&MetadataHandler{}).
				Context()
			foo := new(Foo)
			if transactional, _, err := miruken.Execute[Transactional](handler, foo); err == nil {
				suite.NotNil(transactional)
				suite.Equal(TransactionalRequiresNew, transactional.mode)
				suite.Equal(1, foo.Count())
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Invalid", func() {
			defer func() {
				if r := recover(); r != nil {
					var err *miruken.HandlerInfoError
					if errors.As(r.(error), &err) {
						suite.Equal(
							"unrecognized transactional mode \"suppress\"",
							err.Cause.Error())
					} else {
						suite.Fail("Expected HandlerInfoError")
					}
				}
			}()
			if _, err := setup.New().Specs(&MetadataInvalidHandler{}).Context(); err != nil {
				suite.Fail("unexpected error", err.Error())
			}
		})
	})

	suite.Run("CallSemantics", func() {
		suite.Run("BestEffort", func() {
			ctx, _ := setup.New().Handlers(new(BarHandler)).Context()
			handler := miruken.BuildUp(ctx, miruken.BestEffort)
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(0, foo.Count())
		})

		suite.Run("Broadcast", func() {
			handler, _ := setup.New().
				Specs(&FooHandler{}, &BarHandler{}).
				Handlers(new(FooHandler), new(FooHandler), new(BarHandler)).
				Context()
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())

			result = miruken.BuildUp(handler, miruken.Broadcast).
				Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, foo.Count())
		})
	})

	suite.Run("Intercept", func() {
		suite.Run("Default", func() {
			ctx, _ := setup.New().Specs(&CountByTwoHandler{}).Context()
			handler := miruken.BuildUp(
				ctx,
				miruken.FilterFunc(func(
					callback any,
					greedy bool,
					composer miruken.Handler,
					proceed miruken.ProceedFunc,
				) miruken.HandleResult {
					if cb, ok := callback.(*Foo); ok {
						cb.Inc()
					}
					baz := new(Baz)
					result := composer.Handle(baz, false, nil)
					suite.False(result.IsError())
					suite.Equal(miruken.Handled, result)
					suite.Equal(2, baz.Count())
					return proceed()
				}))
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, foo.Count())

			bar := new(Bar)
			result = handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Count())
		})

		suite.Run("Reentrant", func() {
			ctx, _ := setup.New().Specs(&CountByTwoHandler{}).Context()
			handler := miruken.BuildUp(
				ctx,
				miruken.Reentrant(func(
					callback any,
					greedy bool,
					composer miruken.Handler,
					proceed miruken.ProceedFunc,
				) miruken.HandleResult {
					switch cb := callback.(type) {
					case *Foo:
						cb.Inc()
					case *Baz:
						cb.Inc()
						cb.Inc()
					default:
						return proceed()
					}
					baz := new(Baz)
					result := composer.Handle(baz, false, nil)
					suite.False(result.IsError())
					suite.Equal(miruken.Handled, result)
					suite.Equal(2, baz.Count())
					return proceed()
				}))
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, foo.Count())
		})
	})

	suite.Run("Command", func() {
		handler, _ := setup.New().
			Specs(&CounterHandler{}, &SpecificationHandler{}).
			Context()

		suite.Run("Single", func() {
			suite.Run("Invariant", func() {
				foo := new(Foo)
				_, err := miruken.Command(handler, foo)
				suite.Nil(err)
				suite.Equal(1, foo.Count())
			})
		})

		suite.Run("All", func() {
			suite.Run("Invariant", func() {
				foo := new(Foo)
				_, err := miruken.CommandAll(handler, foo)
				suite.Nil(err)
				suite.Equal(2, foo.Count())
			})
		})
	})

	suite.Run("Execute", func() {
		suite.Run("Single", func() {
			handler, _ := setup.New().
				Specs(&CounterHandler{}).
				Context()

			suite.Run("Invariant", func() {
				if foo, _, err := miruken.Execute[*Foo](handler, new(Foo)); err == nil {
					suite.NotNil(foo)
					suite.Equal(1, foo.Count())
				} else {
					suite.Fail("unexpected error", err.Error())
				}
			})

			suite.Run("Contravariant", func() {
				if foo, _, err := miruken.Execute[any](handler, new(Foo)); err == nil {
					suite.NotNil(foo)
					suite.IsType(&Foo{}, foo)
					suite.Equal(1, foo.(*Foo).Count())
				} else {
					suite.Fail("unexpected error", err.Error())
				}
			})

			suite.Run("BestEffort", func() {
				ctx, _ := setup.New().Handlers(new(BarHandler)).Context()
				handler := miruken.BuildUp(ctx, miruken.BestEffort)
				if foo, _, err := miruken.Execute[*Foo](handler, new(Foo)); err == nil {
					suite.Nil(foo)
				} else {
					suite.Fail("unexpected error", err.Error())
				}
			})

			suite.Run("Mixed", func() {
				handler, _ := setup.New().Specs(&MixedHandler{}).Context()
				if ret, _, err := miruken.Execute[any](handler, new(Foo)); err == nil {
					suite.Equal("Handles *test.Foo", ret)
				} else {
					suite.Fail("unexpected error", err.Error())
				}
				if ret, _, _, err := maps.Out[any](handler, new(Foo)); err == nil {
					suite.Equal("It *test.Foo", ret)
				} else {
					suite.Fail("unexpected error", err.Error())
				}
			})

			suite.Run("Mismatch", func() {
				defer func() {
					if r := recover(); r != nil {
						suite.Equal("reflect.Set: value of type *test.Foo is not assignable to type int", r)
					} else {
						suite.Fail("Expected error")
					}
				}()
				if _, _, err := miruken.Execute[int](handler, new(Foo)); err != nil {
					suite.Fail("unexpected error", err.Error())
				}
			})
		})

		suite.Run("All", func() {
			handler, _ := setup.New().
				Specs(&CountByTwoHandler{}, &SpecificationHandler{}).
				Handlers(&CountByTwoHandler{}, &SpecificationHandler{}).
				Context()

			suite.Run("Invariant", func() {
				if foo, _, err := miruken.ExecuteAll[*Foo](handler, &Foo{Counted{1}}); err == nil {
					suite.NotNil(foo)
					// 1 from explicit return of *CountByTwoHandler
					// 2 from inference of *CountByTwoHandler (1) which includes explicit instance (1)
					suite.Len(foo, 2)
					// 3 from explicit *CountByTwoHandler (2)
					// 4 for inference of *CountByTwoHandler (2)
					// 2 for inference of *SpecificationHandler (1)
					// 9 + 1 = 10 total
					suite.Equal(7, foo[0].Count())
				} else {
					suite.Fail("unexpected error", err.Error())
				}
			})

			suite.Run("Invariant Error", func() {
				handler, _ := setup.New().
					Specs(&CounterHandler{}).
					Context()
				foo := new(Foo)
				foo.Inc()
				foo.Inc()
				if _, _, err := miruken.ExecuteAll[*Foo](handler, foo); err != nil {
					suite.NotNil(err)
					// *CounterHandler returns error based on rule
					suite.Equal("3 is divisible by 3", err.Error())
				} else {
					suite.Fail("expected error")
				}
			})
		})
	})

	suite.Run("Invalid", func() {
		defer func() {
			if r := recover(); r != nil {
				var err *miruken.HandlerInfoError
				if errors.As(r.(error), &err) {
					failures := internal.UnwrapErrors(err.Cause)
					suite.Len(failures, 5)
				} else {
					suite.Fail("Expected HandlerInfoError")
				}
			}
		}()
		_, err := setup.New().Specs(&InvalidHandler{}).Context()
		suite.Nil(err)
		suite.Fail("should cause panic")
	})

	suite.Run("Function Binding", func() {
		suite.Run("Invariant", func() {
			handler, _ := setup.New().Specs(HandleFoo).Context()
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Contravariant", func() {
			handler, _ := setup.New().Specs(HandleCounted).Context()
			bar := new(Bar)
			bar.Inc()
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, bar.Count())
		})

		suite.Run("Invariant Explicit", func() {
			handler, _ := setup.New().
				Specs(HandleFoo).
				Handlers(HandleFoo).
				Context()
			foo := new(Foo)
			result := handler.Handle(foo, true, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			// 1 for explicit instance
			// 1 for inferred call
			suite.Equal(2, foo.Count())
		})

		suite.Run("Contravariant Explicit", func() {
			handler, _ := setup.New().
				Specs(HandleCounted).
				Handlers(HandleCounted).
				Context()
			bar := new(Bar)
			bar.Inc()
			result := handler.Handle(bar, true, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			// Started as 1
			// 2 for explicit instance
			// 2 for inferred instance
			suite.Equal(5, bar.Count())
		})
	})
}

func (suite *HandlesTestSuite) TestHandlesAsync() {
	suite.Run("Simple", func() {
		suite.Run("Promise command", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncHandler{}).Context()
			bar := &Bar{Counted{2}}
			p, err := miruken.Command(handler, bar)
			suite.Nil(err)
			suite.NotNil(p)
			_, err = p.Await()
			suite.Nil(err)
		})

		suite.Run("Promise query", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncHandler{}).Context()
			bar := &Bar{Counted{2}}
			b, p, err := miruken.Execute[*Bar](handler, bar)
			suite.Nil(err)
			suite.Nil(b)
			suite.NotNil(p)
			b, err = p.Await()
			suite.Nil(err)
			suite.Same(bar, b)
			suite.Equal(3, bar.Count())
		})

		suite.Run("Promise dependency", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncHandler{}).Context()
			boo := &Boo{Counted{4}}
			baz, p, err := miruken.Execute[*Baz](handler, boo)
			suite.Nil(err)
			suite.NotNil(p)
			suite.Nil(baz)
			baz, err = p.Await()
			suite.Nil(err)
			suite.Equal(5, boo.Count())
			suite.Equal(1, baz.Count())
		})

		suite.Run("Promise dependency arg", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncHandler{}).Context()
			bam := &Bam{Counted{2}}
			baz, p, err := miruken.Execute[*Baz](handler, bam)
			suite.Nil(err)
			suite.Nil(p)
			suite.NotNil(baz)
			suite.Equal(4, bam.Count())
			suite.Equal(1, baz.Count())
		})

		suite.Run("Promise dependency arg lift", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncHandler{}).Context()
			foo := &Foo{Counted{3}}
			boo, p, err := miruken.Execute[*Boo](handler, foo)
			suite.Nil(err)
			suite.Nil(p)
			suite.NotNil(boo)
			suite.Equal(5, foo.Count())
			suite.Equal(6, boo.Count())
		})
	})

	suite.Run("Complex", func() {
		suite.Run("Promise strict many dependency", func() {
			handler, _ := setup.New().Specs(&ComplexAsyncHandler{}).Context()
			foo := &Foo{Counted{4}}
			baz, p, err := miruken.Execute[[]*Baz](handler, foo)
			suite.Nil(err)
			suite.NotNil(p)
			suite.Nil(baz)
			baz, err = p.Await()
			suite.Nil(err)
			suite.Equal(5, foo.Count())
			suite.Len(baz, 2)
		})
	})

	suite.Run("Errors", func() {
		suite.Run("Reject promise", func() {
			handler, _ := setup.New().Specs(&ErrorAsyncHandler{}).Context()
			foo := new(Foo)
			bar, pb, err := miruken.Execute[*Bar](handler, foo)
			suite.Nil(err)
			suite.Nil(bar)
			suite.NotNil(pb)
			_, err = pb.Await()
			suite.EqualError(err, fmt.Sprintf("bad Foo %p", foo))
		})
	})
}

func TestHandlesTestSuite(t *testing.T) {
	suite.Run(t, new(HandlesTestSuite))
}
