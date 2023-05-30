package maps

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// It maps callbacks bivariantly.
	It struct {
		miruken.CallbackBase
		key     any
		source  any
		match   *Format
	}

	// Builder builds It callbacks.
	Builder struct {
		miruken.CallbackBuilder
		key    any
		source any
	}

	// Strict alias for mapping
	Strict = miruken.Strict
)


func (m *It) Source() any {
	return m.source
}

func (m *It) Key() any {
	in := m.key
	if in == nil {
		in = reflect.TypeOf(m.source)
	}
	out := reflect.TypeOf(m.Target()).Elem()
	return miruken.DiKey{In: in, Out: out}
}

func (m *It) Policy() miruken.Policy {
	return mapsPolicy
}

func (m *It) Matched() *Format {
	return m.match
}

func (m *It) SetMatched(format *Format) {
	m.match = format
}

func (m *It) Dispatch(
	handler  any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	return miruken.DispatchPolicy(handler, m, greedy, composer)
}

func (m *It) String() string {
	return fmt.Sprintf("maps %+v", m.source)
}


// Builder

func (b *Builder) WithKey(
	key any,
) *Builder {
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
}

func (b *Builder) FromSource(
	source any,
) *Builder {
	if miruken.IsNil(source) {
		panic("source cannot be nil")
	}
	b.source = source
	return b
}

func (b *Builder) New() *It {
	return &It{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		source:       b.source,
	}
}

func Out[T any](
	handler     miruken.Handler,
	source      any,
	constraints ...any,
) (t T, tp *promise.Promise[T], m *It, err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.FromSource(source).
		    IntoTarget(&t).
			WithConstraints(constraints...)
	m = builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	} else if _, p := m.Result(false); p != nil {
		tp = promise.Then(p, func(any) T {
			return t
		})
	}
	return
}

func Into[T any](
	handler     miruken.Handler,
	source      any,
	target      *T,
	constraints ...any,
) (p *promise.Promise[any], m *It, err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	var builder Builder
	builder.FromSource(source).
			WithConstraints(constraints...)
	if miruken.TypeOf[T]() == anyType {
		builder.IntoTarget(*target)
	} else {
		builder.IntoTarget(target)
	}
	m = builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	} else {
		_, p = m.Result(false)
	}
	return
}

func Key[T any](
	handler     miruken.Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], m *It, err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.WithKey(key).
		    IntoTarget(&t).
			WithConstraints(constraints...)
	m = builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	} else if _, p := m.Result(false); p != nil {
		tp = promise.Then(p, func(any) T {
			return t
		})
	}
	return
}

func All[T any](
	handler miruken.Handler,
	source      any,
	constraints ...any,
) (t []T, _ *promise.Promise[[]T], _ error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(source) || reflect.TypeOf(source).Kind() != reflect.Slice {
		panic("source must be a non-nil slice")
	}
	ts := reflect.ValueOf(source)
	t   = make([]T, ts.Len())
	var promises []*promise.Promise[T]
	for i := 0; i < ts.Len(); i++ {
		var builder Builder
		builder.FromSource(ts.Index(i).Interface()).
			    IntoTarget(&t[i]).
				WithConstraints(constraints...)
		m := builder.New()
		if result := handler.Handle(m, false, nil); result.IsError() {
			return nil, nil, result.Error()
		} else if !result.Handled() {
			return nil, nil, &miruken.NotHandledError{Callback: m}
		} else if _, p := m.Result(false); p != nil {
			idx := i
			promises = append(promises, promise.Then(p, func(any) T {
				return t[idx]
			}))
		}
	}
	switch len(promises) {
	case 0:
		return
	case 1:
		return nil, promise.Then(promises[0], func(T) []T {
			return t
		}), nil
	default:
		return nil, promise.Then(promise.All(promises...), func([]T) []T {
			return t
		}), nil
	}
}

var (
	mapsPolicy miruken.Policy = &miruken.BivariantPolicy{}
	anyType                   = miruken.TypeOf[any]()
)