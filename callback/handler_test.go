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
) (HandleResult, error) {

	switch foo := callback.(type) {
	case *Foo:
		foo.Handled++
		return context.Handle(Bar{}, false, nil)
	default:
		return NotHandled, nil
	}
}

type BarHandler struct {}

func (h *BarHandler) Handle(
	callback interface{},
	greedy   bool,
	context  HandleContext,
) (HandleResult, error) {

	switch callback.(type) {
	case Bar:
		return Handled, nil
	case *Bar:
		return Handled, nil
	default:
		return NotHandled, nil
	}
}

func TestRootHandler(t *testing.T) {
	foo := Foo{}
	ctx := AddHandlers(
		RootHandler(&FooHandler{}),
		&BarHandler{})

	result, err := ctx.Handle(&foo, false, nil)

	switch {
	case err != nil:
		t.Fatalf("Error:` %v", err)
	case !result.Handled:
		t.Fatalf("Not handled")
	case foo.Handled != 1:
		t.Fatalf("Foo.Handled != 1")
	}
}
