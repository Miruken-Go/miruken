package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"math"
	"testing"
)

type Captured interface {
	Handled() int
	IncHandled(howMany int)
	Composer() miruken.Handler
	SetComposer(composer miruken.Handler)
	Filters()  []miruken.Filter
	AddFilters(filters ...miruken.Filter)
}

type Capture struct {
	handled  int
	composer miruken.Handler
	filters  []miruken.Filter
}

func (c *Capture) Handled() int {
	return c.handled
}

func (c *Capture) IncHandled(howMany int) {
	c.handled += howMany
}

func (c *Capture) Composer() miruken.Handler {
	return c.composer
}

func (c *Capture) SetComposer(composer miruken.Handler) {
	c.composer = composer
}

func (c *Capture) Filters() []miruken.Filter {
	return c.filters
}

func (c *Capture) AddFilters(filters ...miruken.Filter) {
	c.filters = append(c.filters, filters...)
}

type (
	FooC struct { Capture }
	SpecialFooC struct { FooC }
	BarC struct { Capture }
	SpecialBarC struct { BarC }
	BooC struct { Capture }
	BazC struct { Capture }
	SpecialBazC struct {BazC }
	BeeC struct { Capture }
)

type Logging interface {
	Log(msg string)
}

type ConsoleLogger struct{}
func (c *ConsoleLogger) Log(msg string) {
	fmt.Println(msg)
}

// NullFilter test filter
type NullFilter struct {}

func (n *NullFilter) Order() int {
	return math.MaxInt32
}

func (n *NullFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if captured := extractCaptured(ctx.Callback()); captured != nil {
		captured.AddFilters(n)
	}
	return next.Pipe()
}

// LogFilter test filter
type LogFilter struct {}

func (l *LogFilter) Order() int {
	return 1
}

func (l *LogFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	return miruken.DynNext(l, next, ctx, provider)
}

func (l *LogFilter) DynNext(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
	logging  Logging,
)  ([]any, *promise.Promise[[]any], error) {
	captured := extractCaptured(ctx.Callback())
	logging.Log(
		fmt.Sprintf("Log callback %+v", captured))
	if captured != nil {
		captured.AddFilters(l)
	}
	return next.Pipe()
}

// ExceptionFilter test filter
type ExceptionFilter struct {}

func (e *ExceptionFilter) Order() int {
	return 2
}

func (e *ExceptionFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	captured := extractCaptured(ctx.Callback())
	if captured != nil {
		captured.AddFilters(e)
	}
	if result, _, err := next.Pipe(); err != nil {
		return result, nil, err
	} else if _, ok := captured.(*BooC); ok {
		return result, nil, errors.New("system shutdown")
	} else {
		return result, nil, err
	}
}

// AbortFilter test filter
type AbortFilter struct {}

func (a *AbortFilter) Order() int {
	return 0
}

func (a *AbortFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if captured := extractCaptured(ctx.Callback());
		captured == nil || captured.Handled() > 99 {
		return next.Abort()
	}
	return next.Pipe()
}

func extractCaptured(callback miruken.Callback) Captured {
	switch cb := callback.Source().(type) {
	case Captured:
		return cb
	case *miruken.Handles:
		if captured, ok := cb.Source().(Captured); ok {
			return captured
		}
	}
	return nil
}

// FilteringHandler test handler
type FilteringHandler struct {}

func (f FilteringHandler) Order() int {
	return 10
}

func (f FilteringHandler) HandleBar(
	_*struct{
		miruken.Handles
		NullFilter
		LogFilter
		ExceptionFilter `filter:"required"`
		AbortFilter
	  }, bar *BarC,
) {
	bar.IncHandled(1)
}

func (f FilteringHandler) HandleBee(
	_*struct{
		miruken.Handles
		miruken.SkipFilters
		LogFilter
	  },
	bee *BeeC,
) {
	bee.IncHandled(3)
}

func (f FilteringHandler) HandleStuff(
	_*miruken.Handles, callback any,
) {
	if bar, ok := callback.(*BarC); ok {
		bar.IncHandled(-999)
	}
}

func (f FilteringHandler) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if bar, ok := ctx.Callback().Source().(*BarC); ok {
		bar.AddFilters(f)
		bar.IncHandled(1)
	}
	return next.Pipe()
}

// SpecialFilteringHandler test handler
type SpecialFilteringHandler struct {}

func (s SpecialFilteringHandler) HandleFoo(
	_*struct{
		miruken.Handles
		LogFilter
		ExceptionFilter
	  },
	foo *FooC,
) *SpecialFooC {
	return new(SpecialFooC)
}

func (s SpecialFilteringHandler) RemoveBoo(
	_*struct{
		miruken.Handles
		ExceptionFilter
	  },
	boo *BooC,
) {
}

// SingletonHandler test handler

type SingletonHandler struct{}

func (s *SingletonHandler) Constructor(
	_*struct{
		miruken.Provides
		miruken.Singleton
	  },
) {
}

func (s *SingletonHandler) HandleBar(
	_*struct{
		miruken.Handles
		LogFilter
	  },
	bar *BarC,
) {
	bar.IncHandled(3)
}

// SingletonErrorHandler test handler

var errorCount = 0

type SingletonErrorHandler struct {
	count int
}

func (s *SingletonErrorHandler) Constructor(
	_*struct{
		miruken.Provides
		miruken.Singleton
	  },
) error {
	errorCount++
	switch errorCount {
	case 1: return errors.New("something bad")
	case 2: panic("something bad")
	default:
		errorCount = 0
		return nil
	}
}

func (s *SingletonErrorHandler) HandleBee(
	_*miruken.Handles, bee *BeeC,
) {
	bee.IncHandled(3)
}

// BadHandler test handler

type BadHandler struct{}

func (b BadHandler) HandleBar(
	_*struct{
		miruken.Handles
		LogFilter
      },
	bar *BarC,
) {
}

type FilterTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *FilterTestSuite) SetupTest() {
	suite.specs =  []any{
		&FilteringHandler{},
		&SpecialFilteringHandler{},
		&SingletonHandler{},
		&LogFilter{},
		&ConsoleLogger{},
		&ExceptionFilter{},
		&AbortFilter{},
		&NullFilter{},
	}
}

func (suite *FilterTestSuite) Setup() (miruken.Handler, error) {
	return suite.SetupWith(suite.specs...)
}

func (suite *FilterTestSuite) SetupWith(specs ... any) (miruken.Handler, error) {
	return miruken.Setup(miruken.HandlerSpecs(specs...))
}

func (suite *FilterTestSuite) TestFilters() {
	suite.Run("FilterOptions", func () {
		suite.Run("Merges", func () {
			filters  := []miruken.Filter{&NullFilter{}}
			provider := miruken.NewFilterInstanceProvider(false, filters...)
			options  := miruken.FilterOptions{
				Providers:   []miruken.FilterProvider{provider},
				SkipFilters: miruken.Set(true),
			}
			other    := miruken.FilterOptions{}
			other2   := miruken.FilterOptions{
				Providers:   []miruken.FilterProvider{provider},
				SkipFilters: miruken.Set(false),
			}
			miruken.MergeOptions(options, &other)
			suite.Equal(miruken.Set(true), other.SkipFilters)
			suite.ElementsMatch([]miruken.FilterProvider{provider}, options.Providers)
			miruken.MergeOptions(options, &other2)
			suite.Equal(miruken.Set(false), other2.SkipFilters)
			suite.ElementsMatch([]miruken.FilterProvider{provider, provider}, other2.Providers)
		})
	})

	suite.Run("Create Pipeline", func () {
		handler, _ := suite.Setup()
		bar     := new(BarC)
		result  := handler.Handle(bar, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(2, bar.Handled())
		suite.Equal(4, len(bar.Filters()))
		suite.IsType(&LogFilter{}, bar.Filters()[0])
		suite.IsType(&ExceptionFilter{}, bar.Filters()[1])
		suite.IsType(FilteringHandler{}, bar.Filters()[2])
		suite.IsType(&NullFilter{}, bar.Filters()[3])
	})

	suite.Run("Abort Pipeline", func () {
		handler, _ := suite.Setup()
		bar := new(BarC)
		bar.IncHandled(100)
		result := handler.Handle(bar, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(-898, bar.Handled())
	})

	suite.Run("Skip Pipeline", func () {
		suite.Run("Implicit", func() {
			handler, _ := suite.Setup()
			bee := new(BeeC)
			result := handler.Handle(bee, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, bee.Handled())
			suite.Equal(0, len(bee.Filters()))
		})

		suite.Run("Explicit", func() {
			handler, _ := suite.Setup()
			handler  = miruken.BuildUp(handler, miruken.DisableFilters)
			bar     := new(BarC)
			result  := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Handled())
			suite.Equal(2, len(bar.Filters()))
			suite.IsType(&ExceptionFilter{}, bar.Filters()[0])
			suite.IsType(FilteringHandler{}, bar.Filters()[1])
		})
	})

	suite.Run("Singleton", func () {
		suite.Run("Implicit", func() {
			handler, _ := suite.Setup()
			singletonHandler, _, err := miruken.Resolve[*SingletonHandler](handler)
			suite.Nil(err)
			suite.NotNil(singletonHandler)
			singletonHandler2, _, err := miruken.Resolve[*SingletonHandler](handler)
			suite.Nil(err)
			suite.Same(singletonHandler, singletonHandler2)
		})

		suite.Run("Infer", func() {
			handler, _ := suite.SetupWith(
				&SingletonHandler{},
				&ConsoleLogger{},
				&LogFilter{},
			)
			bar := new(BarC)
			bar.IncHandled(10)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(13, bar.Handled())
		})

		suite.Run("Error", func() {
			handler, _ := suite.SetupWith(
				&SingletonErrorHandler{},
			)
			bee := new(BeeC)
			result := handler.Handle(bee, false, nil)
			suite.True(result.IsError())
			suite.Equal("something bad", result.Error().Error())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			result = handler.Handle(bee, false, nil)
			result = handler.Handle(bee, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
		})

		suite.Run("Panic", func() {
			handler, _ := suite.SetupWith(
				&SingletonErrorHandler{},
			)
			bee := new(BeeC)
			result := handler.Handle(bee, false, nil)
			suite.True(result.IsError())
			suite.Equal("something bad", result.Error().Error())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			result = handler.Handle(bee, false, nil)
			suite.True(result.IsError())
			suite.Equal("singleton: panic: something bad", result.Error().Error())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			result = handler.Handle(bee, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
		})
	})

	suite.Run("Missing Dependencies", func () {
		handler, _ := suite.SetupWith(
			&BadHandler{},
			&LogFilter{},
		)
		bar   := new(BarC)
		result := handler.Handle(bar, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
