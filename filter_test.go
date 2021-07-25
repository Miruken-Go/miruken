package miruken

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"math"
	"reflect"
	"testing"
)

type Captured interface {
	Handled() int
	IncHandled(howMany int)
	Composer() Handler
	SetComposer(composer Handler)
	Filters()  []Filter
	AddFilters(filters ... Filter)
}

type Capture struct {
	handled  int
	composer Handler
	filters []Filter
}

func (c *Capture) Handled() int {
	return c.handled
}

func (c *Capture) IncHandled(howMany int) {
	c.handled += howMany
}

func (c *Capture) Composer() Handler {
	return c.composer
}

func (c *Capture) SetComposer(composer Handler) {
	c.composer = composer
}

func (c *Capture) Filters() []Filter {
	return c.filters
}

func (c *Capture) AddFilters(filters ... Filter) {
	c.filters = append(c.filters, filters...)
}

type (
	FooC struct {Capture}
	SpecialFooC struct {Foo}
	BarC struct {Capture}
	SpecialBarC struct {Bar}
	BooC struct {Capture}
	BazC struct {Capture}
	SpecialBazC struct {Baz}
	BeeC struct {Capture}
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

func (n NullFilter) Order() int {
	return math.MaxInt32
}

func (n NullFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	if captured := extractCaptured(context.Callback); captured != nil {
		captured.AddFilters(n)
	}
	return next.Filter()
}

// LogFilter test filter
type LogFilter struct {}

func (l *LogFilter) Order() int {
	return 1
}

func (l *LogFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	return DynNext(l, next, context, provider)
}

func (l *LogFilter) DynNext(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
	logging  Logging,
)  ([]interface{}, error) {
	captured := extractCaptured(context.Callback)
	logging.Log(
		fmt.Sprintf("Log callback %#v", captured))
	if captured != nil {
		captured.AddFilters(l)
	}
	return next.Filter()
}

// ExceptionFilter test filter
type ExceptionFilter struct {}

func (e *ExceptionFilter) Order() int {
	return 2
}

func (e *ExceptionFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	captured := extractCaptured(context.Callback)
	if captured != nil {
		captured.AddFilters(e)
	}
	if result, err := next.Filter(); err != nil {
		return result, err
	} else if _, ok := captured.(*BooC); ok {
		return result, errors.New("system shutdown")
	} else {
		return result, err
	}
}

// AbortFilter test filter
type AbortFilter struct {}

func (a *AbortFilter) Order() int {
	return 0
}

func (a *AbortFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	if captured := extractCaptured(context.Callback);
		captured == nil || captured.Handled() > 99 {
		return next.Abort()
	}
	return next.Filter()
}

func extractCaptured(callback interface{}) Captured {
	switch cb := callback.(type) {
	case Captured: return cb
	case *Command:
		if captured, ok := cb.Callback().(Captured); ok {
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
	_ *struct{
		Handles
		NullFilter
		LogFilter
		ExceptionFilter
		AbortFilter
	},
	bar *BarC,
) {
	bar.IncHandled(1)
}

func (f FilteringHandler) HandleBee(
	_ *struct{
		Handles `bind:"skipFilters"`
		LogFilter
	},
	bee *BeeC,
) {

}

func (f FilteringHandler) HandleStuff(
	_ Handles,
	callback interface{},
) {
	if bar, ok := callback.(*BarC); ok {
		bar.IncHandled(-99)
	}
}

func (f FilteringHandler) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	if bar, ok := context.Callback.(*BarC); ok {
		bar.AddFilters(f)
		bar.IncHandled(1)
	}
	return next.Filter()
}

// SpecialFilteringHandler test handler
type SpecialFilteringHandler struct {}

func (s SpecialFilteringHandler) HandleFoo(
	_ *struct{
		Handles
		LogFilter
		ExceptionFilter
	},
	foo *FooC,
) *SpecialFooC {
	return new(SpecialFooC)
}

func (s SpecialFilteringHandler) RemoveBoo(
	_ *struct{
	Handles
	ExceptionFilter
},
	boo *BooC,
) {
}

type FilterTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *FilterTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		reflect.TypeOf((*FilteringHandler)(nil)),
		reflect.TypeOf((*SpecialFilteringHandler)(nil)),
		reflect.TypeOf((*LogFilter)(nil)),
		reflect.TypeOf((*ConsoleLogger)(nil)),
		reflect.TypeOf((*ExceptionFilter)(nil)),
		reflect.TypeOf((*AbortFilter)(nil)),
		reflect.TypeOf((*NullFilter)(nil)).Elem(),
	}
	suite.HandleTypes = handleTypes
}

func (suite *FilterTestSuite) InferenceRoot() Handler {
	return NewRootHandler(WithHandlerTypes(suite.HandleTypes...))
}

func (suite *FilterTestSuite) TestFilters() {
	suite.Run("FilterOptions", func () {
		suite.Run("Merges", func () {
			filters  := []Filter{NullFilter{}}
			provider := &filterInstanceProvider{filters, false}
			options  := FilterOptions{
				Providers:   []FilterProvider{provider},
				SkipFilters: OptionTrue,
			}
			other    := FilterOptions{}
			other2   := FilterOptions{
				Providers:   []FilterProvider{provider},
				SkipFilters: OptionFalse,
			}
			MergeOptions(options, &other)
			suite.True(other.SkipFilters.Bool())
			suite.ElementsMatch([]FilterProvider{provider}, options.Providers)
			MergeOptions(options, &other2)
			suite.False(other2.SkipFilters.Bool())
			suite.ElementsMatch([]FilterProvider{provider, provider}, other2.Providers)
		})

		suite.Run("Create", func () {
			handler := suite.InferenceRoot()
			bar     := new(BarC)
			result  := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(Handled, result)
			suite.Equal(2, bar.Handled())
			suite.Equal(4, len(bar.Filters()))
			suite.IsType(&LogFilter{}, bar.Filters()[0])
			suite.IsType(&ExceptionFilter{}, bar.Filters()[1])
			suite.IsType(FilteringHandler{}, bar.Filters()[2])
			suite.IsType(NullFilter{}, bar.Filters()[3])
		})
	})
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
