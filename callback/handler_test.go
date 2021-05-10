package callback

import (
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

func (h *BarHandler) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) HandleResult {

	switch callback.(type) {
	case Bar:
		return Handled
	case *Bar:
		return Handled
	default:
		return NotHandled
	}
}

func TestRootHandler(t *testing.T) {
	foo := Foo{}
	ctx := AddHandlers(
		RootHandler(&FooHandler{}),
		&BarHandler{})

	result := ctx.Handle(&foo, false, nil)

	switch {
	case result.IsError():
		t.Fatalf("Error:` %v", result.Error())
	case !result.IsHandled():
		t.Fatalf("Not handled")
	case foo.Handled != 1:
		t.Fatalf("Foo.handled != 1")
	}
}
