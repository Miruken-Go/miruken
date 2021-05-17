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
	ctx.Handle(new(Bar), false, nil)
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

// EverythingHandler

type EverythingHandler struct{}

func (h *EverythingHandler) HandleEverything(
	policy   Handles,
	callback interface{},
) HandleResult {
	switch f := callback.(type) {
	case *Foo:
		f.Inc()
		return Handled
	case Counter:
		f.Inc()
		f.Inc()
		return Handled
	default:
		return NotHandled
	}
}

// SpecificationHandler

type SpecificationHandler struct{}

func (h *SpecificationHandler) HandleFoo(
	binding *struct {
		Handles  `strict:"true"`
	},
	foo *Foo,
) HandleResult {
	foo.Inc()
	return Handled
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
}

func (suite *HandlerTestSuite) SetupTest() {
}

func (suite *HandlerTestSuite) TestHandles() {
	suite.Run("Invariant", func () {
		ctx    := NewHandleContext(WithHandlers(new(FooHandler), new(BarHandler)))
		foo    := new(Foo)
		result := ctx.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		ctx    := NewHandleContext(WithHandlers(new(CounterHandler)))
		foo    := new(Foo)
		result := ctx.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(1, foo.Count())
	})
}

func (suite *HandlerTestSuite) TestHandlesResult() {
	ctx := NewHandleContext(WithHandlers(new(CounterHandler)))

	suite.Run("Handled", func() {
		foo := new(Foo)
		foo.Inc()
		result := ctx.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(NotHandled, result)
	})

	suite.Run("NotHandled", func() {
		foo := new(Foo)
		foo.Inc()
		foo.Inc()
		result := ctx.Handle(foo, false, nil)
		suite.True(result.IsError())
		suite.Equal("3 is divisible by 3", result.Error().Error())
	})
}

func (suite *HandlerTestSuite) TestHandlesMultiple() {
	multi := new(MultiHandler)
	ctx   := NewHandleContext(WithHandlers(multi))
	foo   := new(Foo)
	for i := 0; i < 4; i++ {
		result := ctx.Handle(foo, false, nil)
		suite.Equal(Handled, result)
		suite.Equal(i + 1, foo.Count())
	}

	suite.Equal(4, multi.foo.Count())
	suite.Equal(4, multi.bar.Count())

	result := ctx.Handle(foo, false, nil)
	suite.True(result.IsError())
	suite.Equal("count reached 5", result.Error().Error())

	suite.Equal(5, multi.foo.Count())
	suite.Equal(4, multi.bar.Count())
}

func (suite *HandlerTestSuite) TestHandlesSpecification() {
	ctx := NewHandleContext(WithHandlers(new(SpecificationHandler)))

	suite.Run("Strict", func() {
		foo    := new(Foo)
		result := ctx.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(1, foo.Count())
	})
}

func (suite *HandlerTestSuite) TestHandlesEverything() {
	ctx := NewHandleContext(WithHandlers(new(EverythingHandler)))

	suite.Run("Invariant", func () {
		foo    := new(Foo)
		result := ctx.Handle(foo, false, nil)

		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		bar    := new(Bar)
		result := ctx.Handle(bar, false, nil)

		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(2, bar.Count())
	})
}

func (suite *HandlerTestSuite) TestInvokesWithResult() {
	ctx := NewHandleContext(WithHandlers(new(CounterHandler)))

	suite.Run("Invariant", func () {
		var foo *Foo
		if err := Invoke(ctx, new(Foo), &foo); err == nil {
			suite.NotNil(*foo)
			suite.Equal(1, foo.Count())
		} else {
			suite.Failf("unexpected error: %v", err.Error())
		}
	})

	suite.Run("Covariant", func () {
		var foo interface{}
		if err := Invoke(ctx, new(Foo), &foo); err == nil {
			suite.NotNil(foo)
			suite.IsType(&Foo{}, foo)
			suite.Equal(1, foo.(*Foo).Count())
		} else {
			suite.Failf("unexpected error: %v", err.Error())
		}
	})
}

func (suite *HandlerTestSuite) TestInvokesAllWithResult() {
	ctx := NewHandleContext(WithHandlers(
		new(CounterHandler), new(SpecificationHandler)))

	suite.Run("Invariant", func () {
		var foo []*Foo
		if err := InvokeAll(ctx, new(Foo), &foo); err == nil {
			suite.NotNil(foo)
			suite.Len(foo, 1)
			suite.Equal(2, foo[0].Count())
		} else {
			suite.Failf("unexpected error: %v", err.Error())
		}
	})
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

	NewHandleContext(WithHandlers(new(InvalidHandler)))
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}