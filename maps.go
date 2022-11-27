package miruken

import (
	"errors"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
)

type (
	// Maps callbacks bivariantly.
	Maps struct {
		CallbackBase
		key    any
		source any
		target any
	}

	// MapsBuilder builds Maps callbacks.
	MapsBuilder struct {
		CallbackBuilder
		key    any
		source any
		target any
	}

	// FormatDirection indicates direction of formatting.
	FormatDirection uint8

	// Format is a BindingConstraint for applying formatting.
	Format struct {
		Direction FormatDirection
		Tag       any
	}
)

const (
	FormatDirectionNone FormatDirection = 0
	FormatDirectionTo FormatDirection = 1 << iota
	FormatDirectionFrom
)

// Maps

func (m *Maps) Source() any {
	return m.source
}

func (m *Maps) Target() any {
	return m.target
}

func (m *Maps) Key() any {
	in := m.key
	if in == nil {
		in = reflect.TypeOf(m.source)
	}
	out := reflect.TypeOf(m.target).Elem()
	return DiKey{in, out}
}

func (m *Maps) Policy() Policy {
	return _mapsPolicy
}

func (m *Maps) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, m, greedy, composer)
}


// Format

func (f *Format) InitWithTag(tag reflect.StructTag) error {
	var format string
	if to, ok := tag.Lookup("to"); ok {
		format      = to
		f.Direction = FormatDirectionTo
	} else if from, ok := tag.Lookup("from"); ok {
		format      = from
		f.Direction = FormatDirectionFrom
	}
	if format = strings.TrimSpace(format); len(format) > 0 {
		f.Tag = format
	}
	if IsNil(f.Tag) {
		return ErrFormatMissing
	}
	return nil
}

func (f *Format) Merge(constraint BindingConstraint) bool {
	if format, ok := constraint.(*Format); ok {
		f.Direction = format.Direction
		f.Tag       = format.Tag
		return true
	}
	return false
}

func (f *Format) Require(metadata *BindingMetadata) {
	if tag := f.Tag; !IsNil(tag) {
		metadata.Set(_formatType, f)
	}
}

func (f *Format) Matches(metadata *BindingMetadata) bool {
	if m, ok := metadata.Get(_formatType); ok {
		if format, ok := m.(*Format); ok {
			return *format == *f
		}
	}
	return false
}

// To maps to a format.
func To(format string) *Format {
	return &Format{FormatDirectionTo, format}
}

// From maps from a format.
func From(format string) *Format {
	return &Format{FormatDirectionFrom, format}
}

// MapsBuilder

func (b *MapsBuilder) WithKey(
	key any,
) *MapsBuilder {
	if IsNil(key) {
		panic("key cannot be nil")
	}
	b.key = key
	return b
}

func (b *MapsBuilder) FromSource(
	source any,
) *MapsBuilder {
	if IsNil(source) {
		panic("source cannot be nil")
	}
	b.source = source
	return b
}

func (b *MapsBuilder) ToTarget(
	target any,
) *MapsBuilder {
	if IsNil(target) {
		panic("source cannot be nil")
	}
	b.target = target
	return b
}

func (b *MapsBuilder) NewMaps() *Maps {
	return &Maps{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		source:       b.source,
		target:       b.target,
	}
}

func Map[T any](
	handler         Handler,
	source          any,
	constraints ... any,
) (t T, tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder MapsBuilder
	builder.FromSource(source).
		    ToTarget(&t).
			WithConstraints(constraints...)
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = &NotHandledError{maps}
	} else {
		_, tp, err = CoerceResult[T](maps, &t)
	}
	return
}

func MapInto[T any](
	handler         Handler,
	source          any,
	target          *T,
	constraints ... any,
) (tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	var builder MapsBuilder
	builder.FromSource(source).
			ToTarget(target).
			WithConstraints(constraints...)
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = &NotHandledError{maps}
	} else {
		_, tp, err = CoerceResult[T](maps, target)
	}
	return
}

func MapKey[T any](
	handler         Handler,
	key             any,
	constraints ... any,
) (t T, tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	var builder MapsBuilder
	builder.WithKey(key).
			ToTarget(&t).
			WithConstraints(constraints...)
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = &NotHandledError{maps}
	} else {
		_, tp, err = CoerceResult[T](maps, &t)
	}
	return
}

func MapAll[T any](
	handler         Handler,
	source          any,
	constraints ... any,
) (t []T, _ *promise.Promise[[]T], _ error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if IsNil(source) || reflect.TypeOf(source).Kind() != reflect.Slice {
		panic("source must be a non-nil slice")
	}
	ts := reflect.ValueOf(source)
	t   = make([]T, ts.Len())
	var promises []*promise.Promise[T]
	for i := 0; i < ts.Len(); i++ {
		var builder MapsBuilder
		builder.FromSource(ts.Index(i).Interface()).
				ToTarget(&t[i]).
				WithConstraints(constraints...)
		maps := builder.NewMaps()
		if result := handler.Handle(maps, false, nil); result.IsError() {
			return nil, nil, result.Error()
		} else if !result.handled {
			return nil, nil, &NotHandledError{maps}
		}
		if _, pm, err := CoerceResult[T](maps, &t[i]); err != nil {
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

var (
	_mapsPolicy Policy = &BivariantPolicy{}
	_formatType        = TypeOf[*Format]()
	ErrFormatMissing   = errors.New("the Format constraint requires a non-empty tag")
)