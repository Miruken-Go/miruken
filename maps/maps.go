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
) (t T, tp *promise.Promise[T], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.FromSource(source).
		    ToTarget(&t).
			WithConstraints(constraints...)
	m := builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	} else {
		tp, err = ensureTargetWritten[T](&t, m)
	}
	return
}

func Into[T any](
	handler     miruken.Handler,
	source      any,
	target      *T,
	constraints ...any,
) (vp *promise.Promise[promise.Void], err error) {
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
		builder.ToTarget(*target)
	} else {
		builder.ToTarget(target)
	}
	m := builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	}else if m.TargetWritten() {
		vp, err = miruken.CompleteResult(m)
	} else {
		var tp *promise.Promise[T]
		if _, tp, err = miruken.CoerceResult[T](m, target); err == nil && tp != nil {
			vp = promise.Then(tp, func(T) promise.Void {
				return promise.Void{}
			})
		}
	}
	return
}

func Key[T any](
	handler     miruken.Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder Builder
	builder.WithKey(key).
			ToTarget(&t).
			WithConstraints(constraints...)
	m := builder.New()
	if result := handler.Handle(m, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: m}
	} else {
		tp, err = ensureTargetWritten[T](&t, m)
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
				ToTarget(&t[i]).
				WithConstraints(constraints...)
		m := builder.New()
		if result := handler.Handle(m, false, nil); result.IsError() {
			return nil, nil, result.Error()
		} else if !result.Handled() {
			return nil, nil, &miruken.NotHandledError{Callback: m}
		}
		if pm, err := ensureTargetWritten[T](&t[i], m); err != nil {
			return nil, nil, err
		} else if pm != nil {
			promises = append(promises, pm)
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

func ensureTargetWritten[T any](
	t *T,
	m *It,
) (tp *promise.Promise[T], err error) {
	if m.TargetWritten() {
		var vp *promise.Promise[promise.Void]
		if vp, err = miruken.CompleteResult(m); err == nil && vp != nil {
			tp = promise.Then(vp, func(promise.Void) T {
				return *t
			})
		}
	} else {
		_, tp, err = miruken.CoerceResult[T](m, t)
	}
	return
}

var (
	mapsPolicy miruken.Policy = &miruken.BivariantPolicy{}
	anyType                   = miruken.TypeOf[any]()
)