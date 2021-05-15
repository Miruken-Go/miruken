package miruken

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"io"
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

type Bar struct {
	Counted
}

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

func (h *FooHandler) doSomething() {

}

// BarHandler

type BarHandler struct {}

func (h *BarHandler) HandleBar(
	policy Handles,
	bar    Bar,
) {
	fmt.Printf("Handled %#v\n", bar)
}

// CounterHandler

type CounterHandler struct {}

func (h *CounterHandler) HandleCounted(
	policy  Handles,
	counter Counter,
) (Counter, HandleResult) {
	switch c := counter.Inc(); {
	case c % 3 == 0:
		err := fmt.Errorf("%v is divisible by 3", c)
		return nil, NotHandled.WithError(err)
	case c % 2 == 0: return nil, NotHandled
	default: return counter, Handled
	}
}

// MultiHandler

type MultiHandler struct {
	foo Foo
	bar Bar
}

func (h *MultiHandler) HandleFoo(
	policy Handles,
	foo    *Foo,
	ctx    HandleContext,
) error {
	h.foo.Inc()
	if foo.Inc() == 5 {
		return errors.New("count reached 5")
	}
	ctx.Handle(&Bar{}, false, nil)
	return nil
}

func (h *MultiHandler) HandleBar(
	policy Handles,
	bar    *Bar,
) HandleResult {
	h.bar.Inc()
	if bar.Inc() % 2 == 0 {
		return Handled
	}
	return NotHandled
}

// InvalidHandler

type InvalidHandler struct {}

func (h *InvalidHandler) MissingCallback(
	policy Handles,
) {
}

func (h *InvalidHandler) AdditionalDependencies(
	policy Handles,
	foo    *Foo,
	reader io.Reader,
) {
}

func (h *InvalidHandler) TooManyReturnValues(
	policy Handles,
	bar    *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (h *InvalidHandler) SecondReturnMustBeErrorOrHandleResult(
	policy   Handles,
	counter *Counter,
) (Foo, string) {
	return Foo{}, "bad"
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

func (suite *HandlerTestSuite) TestHandlesMultiple() {
	multi := &MultiHandler{}
	ctx   := NewHandleContext(WithHandlers(multi))
	foo   := Foo{}
	for i := 0; i < 4; i++ {
		result := ctx.Handle(&foo, false, nil)
		suite.Equal(Handled, result)
		suite.Equal(i + 1, foo.Count())
	}

	suite.Equal(4, multi.foo.Count())
	suite.Equal(4, multi.bar.Count())

	result := ctx.Handle(&foo, false, nil)
	suite.True(result.IsError())
	suite.Equal("count reached 5", result.Error().Error())

	suite.Equal(5, multi.foo.Count())
	suite.Equal(4, multi.bar.Count())
}

func (suite *HandlerTestSuite) TestInvalidHandler() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(*HandlerDescriptorError); ok {
				failures := 0
				var errMethod *MethodBindingError
				for reason := errors.Unwrap(err.Reason);
					errors.As(reason, &errMethod); reason = errors.Unwrap(reason) {
						failures++
				}
				suite.Equal(4, failures)
			} else {
				suite.Fail("Expected HandlerDescriptorError")
			}
		}
	}()

	NewHandleContext(WithHandlers(&InvalidHandler{}))
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}