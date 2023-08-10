package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/stretchr/testify/suite"
	"math"
	"testing"
)

type (
	Captured interface {
		Handled() int
		IncHandled(howMany int)
		Composer() miruken.Handler
		SetComposer(composer miruken.Handler)
		Filters()  []miruken.Filter
		AddFilters(filters ...miruken.Filter)
	}

	Capture struct {
		handled  int
		composer miruken.Handler
		filters  []miruken.Filter
	}
)


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
	FooC        struct { Capture }
	SpecialFooC struct { FooC }
	BarC        struct { Capture }
	BooC        struct { Capture }
	BeeC        struct { Capture }

	Logging interface {
		Log(msg string)
	}

	ConsoleLogger struct{}
)


func (c *ConsoleLogger) Log(msg string) {
	fmt.Println(msg)
}


type (
	NullFilter struct {}
	LogFilter struct { miruken.FilterAdapter }
	ExceptionFilter struct {}
	AbortFilter struct {}
)


func (n *NullFilter) Order() int {
	return math.MaxInt32
}

func (n *NullFilter) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if captured := extractCaptured(ctx.Callback); captured != nil {
		captured.AddFilters(n)
	}
	return next.Pipe()
}


func (l *LogFilter) Order() int {
	return 1
}

func (l *LogFilter) Log(
	next    miruken.Next,
	ctx     miruken.HandleContext,
	logging Logging,
)  ([]any, *promise.Promise[[]any], error) {
	captured := extractCaptured(ctx.Callback)
	logging.Log(
		fmt.Sprintf("Log callback %+v", captured))
	if captured != nil {
		captured.AddFilters(l)
	}
	return next.Pipe()
}


func (e *ExceptionFilter) Order() int {
	return 2
}

func (e *ExceptionFilter) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	captured := extractCaptured(ctx.Callback)
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


func (a *AbortFilter) Order() int {
	return 0
}

func (a *AbortFilter) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	if captured := extractCaptured(ctx.Callback);
		captured == nil || captured.Handled() > 99 {
		return next.Abort()
	}
	return next.Pipe()
}


func extractCaptured(callback miruken.Callback) Captured {
	switch cb := callback.Source().(type) {
	case Captured:
		return cb
	case *handles.It:
		if captured, ok := cb.Source().(Captured); ok {
			return captured
		}
	}
	return nil
}


type (
	FilteringHandler struct {}
	SpecialFilteringHandler struct {}
	SingletonHandler struct{}
)


func (h FilteringHandler) Order() int {
	return 10
}

func (h FilteringHandler) HandleBar(
	_*struct{
		handles.It
		NullFilter
		LogFilter
		ExceptionFilter `filter:"required"`
		AbortFilter
	  }, bar *BarC,
	cfg map[string]any,
) {
	bar.IncHandled(1)
	fmt.Println(cfg)
}

func (h FilteringHandler) HandleBee(
	_*struct{
		handles.It
		miruken.SkipFilters
		LogFilter
	  },
	bee *BeeC,
	cfg map[string]any,
) {
	bee.IncHandled(3)
	fmt.Println(cfg)
}

func (h FilteringHandler) HandleStuff(
	_ *handles.It, callback any,
) {
	if bar, ok := callback.(*BarC); ok {
		bar.IncHandled(-999)
	}
}

func (h FilteringHandler) Next(
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	if bar, ok := ctx.Callback.Source().(*BarC); ok {
		bar.AddFilters(h)
		bar.IncHandled(1)
	}
	return next.Pipe(map[string]any{
		"foo": "bar",
	})
}


func (s SpecialFilteringHandler) HandleFoo(
	_*struct{
		handles.It
		LogFilter
		ExceptionFilter
	  },
	foo *FooC,
	data map[string]any,
) map[string]any {
	foo.IncHandled(1)
	data["more"] = "stuff"
	return data
}

func (s SpecialFilteringHandler) HandleBee(
	_*struct{
		handles.It
		ExceptionFilter
	 },
	bee *BeeC,
	data map[string]any,
) map[string]any {
	bee.IncHandled(1)
	data["hello"] = "world"
	return data
}

func (s SpecialFilteringHandler) RemoveBoo(
	_*struct{
		handles.It
		ExceptionFilter
	  },
	_ *BooC,
) {
}

func (s SpecialFilteringHandler) Load(
	next miruken.Next,
) ([]any, *promise.Promise[[]any], error) {
	return next.Pipe(map[string]any{
		"foo": "bar",
	})
}

func (s SpecialFilteringHandler) LoadFoo(
	foo  *FooC,
	next miruken.Next,
	data map[string]any,
) ([]any, *promise.Promise[[]any], error) {
	data["callback"] = foo
	return next.Pipe(data)
}

func (s *SingletonHandler) Constructor(
	_*struct{
		provides.It
		provides.Single
	  },
) {
}

func (s *SingletonHandler) HandleBar(
	_*struct{
		handles.It
		LogFilter
	  },
	bar *BarC,
) {
	bar.IncHandled(3)
}


var errorCount = 0

type (
	SingletonErrorHandler struct {
		count int
	}
	BadHandler struct{}
)

func (s *SingletonErrorHandler) Constructor(
	_*struct{
		provides.It
		provides.Single
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
	_ *handles.It, bee *BeeC,
) {
	bee.IncHandled(3)
}


func (b BadHandler) HandleBar(
	_*struct{
		handles.It
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

func (suite *FilterTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup().Specs(specs...).Handler()
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
			other  := miruken.FilterOptions{}
			other2 := miruken.FilterOptions{
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
		bar    := new(BarC)
		result := handler.Handle(bar, false, nil)
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

		suite.Run("Override", func() {
			handler, _ := suite.Setup()
			handler  = miruken.BuildUp(handler, miruken.DisableFilters, miruken.EnableFilters)
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
	})

	suite.Run("Single", func () {
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
			handler, _ := suite.Setup(
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
			handler, _ := suite.Setup(
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
			handler, _ := suite.Setup(
				&SingletonErrorHandler{},
			)
			bee := new(BeeC)
			result := handler.Handle(bee, false, nil)
			suite.True(result.IsError())
			suite.Equal("something bad", result.Error().Error())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			result = handler.Handle(bee, false, nil)
			suite.True(result.IsError())
			suite.Equal("single: panic: something bad", result.Error().Error())
			suite.Equal(miruken.NotHandledAndStop, result.WithoutError())
			result = handler.Handle(bee, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
		})
	})

	suite.Run("Compound", func () {
		suite.Run("ApplyAll", func() {
			handler, _ := suite.Setup(
				&SpecialFilteringHandler{},
				&LogFilter{},
				&ConsoleLogger{},
				&ExceptionFilter{},
			)
			foo := new(FooC)
			r, _, err := miruken.Execute[map[string]any](handler, foo)
			suite.Nil(err)
			suite.NotNil(r)
			suite.Equal(1, foo.Handled())
			suite.Equal("bar", r["foo"])
			suite.Equal("stuff", r["more"])
			suite.Same(foo, r["callback"])
		})

		suite.Run("ApplyTo", func() {
			handler, _ := suite.Setup(
				&SpecialFilteringHandler{},
				&LogFilter{},
				&ConsoleLogger{},
				&ExceptionFilter{},
			)
			bee := new(BeeC)
			r, _, err := miruken.Execute[map[string]any](handler, bee)
			suite.Nil(err)
			suite.NotNil(r)
			suite.Equal(1, bee.Handled())
			suite.Equal("bar", r["foo"])
			suite.Equal("world", r["hello"])
			suite.NotContains(r, "callback")
		})
	})

	suite.Run("Missing Dependencies", func () {
		handler, _ := suite.Setup(
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
