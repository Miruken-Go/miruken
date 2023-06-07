package provides

import (
	"github.com/miruken-go/miruken"
	"reflect"
)


// For matches dependencies against a receiver hierarchy.
type For[T any] struct {
	typ reflect.Type
}


// For

func (f *For[T]) Init() error {
	if f.typ = miruken.TypeOf[T](); f.typ.Kind() == reflect.Ptr {
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

func (f *For[T]) Satisfies(
	required miruken.Constraint,
	callback miruken.Callback,
) bool {
	if required != nil {
		return false
	}
	if p, ok := callback.(*It); ok {
		return f.matches(p.Parent())
	}
	return true
}

func (f *For[T]) matches(p *It) bool {
	for p != nil {
		if b := p.Binding(); b != nil {
			if typ := b.LogicalOutputType(); typ != nil {
				if typ.Kind() == reflect.Ptr {
					typ = typ.Elem()
				}
				if typ.AssignableTo(f.typ) {
					return true
				}
				p = p.Parent()
			}
		}
	}
	return false
}
