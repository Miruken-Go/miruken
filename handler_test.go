package miruken

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
)

type Counter interface {
	Count() int
	Inc() int
}

type Counted struct {
	count int
}

func (c *Counted) Count() int {
	return c.count
}

func (c *Counted) Inc() int {
	c.count++
	return c.count
}

type Foo struct {
	Counted
}

type Bar struct {}

// FooHandler

type FooHandler struct {}

func (h *FooHandler) Handle(
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
) HandleResult {

	switch foo := callback.(type) {
	case *Foo:
		foo.Inc()
		return ctx.Handle(Bar{}, false, nil)
	default:
		return NotHandled
	}
}

// BarHandler

type BarHandler struct {}

func (h *BarHandler) HandleBar(
	policy Handles,
	bar    Bar,
	ctx    HandleContext,
) {
	fmt.Printf("Handled %#v\n", bar)
}

// CounterHandler

type CounterHandler struct {}

func (h *CounterHandler) HandleCounted(
	policy  Handles,
	counter Counter,
	ctx     HandleContext,
) HandleResult {
	switch c := counter.Inc(); {
	case c % 3 == 0:
		err := fmt.Errorf("%v is divisible by 3", c)
		return NotHandled.WithError(err)
	case c % 2 == 0: return NotHandled
	default: return Handled
	}
}

type HandlerTestSuite struct {
	suite.Suite
	ctx HandleContext
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.ctx = NewHandleContext(
		WithHandlers(&FooHandler{}, &BarHandler{}))
}

func (suite *HandlerTestSuite) TestHandlesInvariant() {
	foo    := Foo{}
	result := suite.ctx.Handle(&foo, false, nil)

	suite.False(result.IsError())
	suite.Equal(Handled, result)
	suite.Equal(1, foo.Count())
}

func (suite *HandlerTestSuite) TestHandlesContravariant() {
	ctx    := NewHandleContext(WithHandlers(&CounterHandler{}))
	foo    := Foo{}
	result := ctx.Handle(&foo, false, nil)

	suite.False(result.IsError())
	suite.Equal(Handled, result)
	suite.Equal(1, foo.Count())
}

func (suite *HandlerTestSuite) TestHandlesExplicitResult() {
	ctx    := NewHandleContext(WithHandlers(&CounterHandler{}))
	foo    := Foo{}
	foo.Inc()
	result := ctx.Handle(&foo, false, nil)

	suite.False(result.IsError())
	suite.Equal(NotHandled, result)

	result = ctx.Handle(&foo, false, nil)
	suite.True(result.IsError())
	suite.Equal("3 is divisible by 3", result.Error().Error())
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}