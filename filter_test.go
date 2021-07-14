package miruken

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"math"
	"testing"
)

type Captured interface {
	Handled()  int
	IncHandled()
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

func (c *Capture) IncHandled() {
	c.handled++
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
)

type Logging interface {
	Log(msg string)
}

type ConsoleLogger struct{}
func (c *ConsoleLogger) Log(msg string) {
	fmt.Println(msg)
}

type NullFilter struct {}

func (n NullFilter) Order() int {
	return math.MaxInt32
}

func (n NullFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	return next.Filter()
}

type LogFilter struct {}

func (l *LogFilter) Order() int {
	return 1
}

func (l *LogFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]interface{}, error) {
	//captured := extractCaptured(context.Callback)
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

type FilterTestSuite struct {
	suite.Suite
}

func (suite *FilterTestSuite) TestFilters() {
	suite.Run("Foo", func () {

	})
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
