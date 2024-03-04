package provides

import (
	"reflect"

	"github.com/miruken-go/miruken"
)

type (
	// For matches dependencies against a parent receiver.
	For[T any] struct {
		typ   reflect.Type
		graph bool
	}

	// ForGraph matches dependencies against a receiver hierarchy.
	ForGraph[T any] struct {
		For[T]
	}
)

// For

func (f *For[T]) Init() error {
	if f.typ = reflect.TypeFor[T](); f.typ.Kind() == reflect.Ptr {
		f.typ = f.typ.Elem()
	}
	return nil
}

func (f *For[T]) Required() bool {
	return true
}

func (f *For[T]) Implied() bool {
	return true
}

func (f *For[T]) Satisfies(required miruken.Constraint, ctx miruken.HandleContext) bool {
	if required != nil {
		return false
	}
	if p, ok := ctx.Callback.(*It); ok {
		return f.matches(p.Parent(), f.graph)
	}
	return true
}

func (f *For[T]) matches(p *It, graph bool) bool {
	for p != nil {
		if b := p.Binding(); b != nil {
			if typ := b.LogicalOutputType(); typ != nil {
				if typ.Kind() == reflect.Ptr {
					typ = typ.Elem()
				}
				if typ.AssignableTo(f.typ) {
					return true
				}
				if graph {
					p = p.Parent()
				} else {
					return false
				}
			}
		}
	}
	return false
}

// ForGraph

func (f *ForGraph[T]) Init() error {
	f.For.graph = true
	return f.For.Init()
}
