package callback

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type Foo struct{
	Handled int
}

type Bar struct{}

type FooHandler struct {}

func (h *FooHandler) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {

	switch foo := callback.(type) {
	case *Foo:
		foo.Handled++
		return context.Handle(Bar{}, false, nil)
	default:
		return NotHandled
	}
}

type BarHandler struct {}

func (h *BarHandler) HandleBar(
	policy   Handles,
	bar      Bar,
	context  HandleContext,
) {

}

type HandlerTestSuite struct {
	suite.Suite
	ctx HandleContext
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.ctx = WithHandlerDescriptorFactory(
		NewHandleContext(&FooHandler{}, &BarHandler{}),
		NewMutableHandlerDescriptorFactory())
}

func (suite *HandlerTestSuite) TestHandle() {
	foo    := Foo{}
	result := suite.ctx.Handle(&foo, false, nil)

	suite.False(result.IsError())
	suite.Equal(NotHandled, result)
	suite.Equal(1, foo.Handled)
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}