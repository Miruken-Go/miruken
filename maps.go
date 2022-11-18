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
		source any
		target any
		format any
	}

	// Format is the BindingConstraint for matching formats.
	Format struct {
		as any
	}

	// MapsBuilder builds Maps callbacks.
	MapsBuilder struct {
		CallbackBuilder
		source any
		target any
		format any
	}
)


// Maps

func (m *Maps) Source() any {
	return m.source
}

func (m *Maps) Target() any {
	return m.target
}

func (m *Maps) Format() any {
	return m.format
}

func (m *Maps) Key() any {
	in  := reflect.TypeOf(m.source)
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
	if as, ok := tag.Lookup("as"); ok {
		if format := strings.TrimSpace(as); len(format) > 0 {
			f.as = format
		}
	}
	if IsNil(f.as){
		return ErrFormatMissing
	}
	return nil
}

func (f *Format) Merge(constraint BindingConstraint) bool {
	if format, ok := constraint.(*Format); ok {
		f.as = format.as
		return true
	}
	return false
}

func (f *Format) Require(metadata *BindingMetadata) {
	if as := f.as; !IsNil(as) {
		metadata.Set(_formatType, as)
	}
}

func (f *Format) Matches(metadata *BindingMetadata) bool {
	if format, ok := metadata.Get(_formatType); ok {
		return format == f.as
	}
	return false
}


// MapsBuilder

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

func (b *MapsBuilder) WithFormat(
	format any,
) *MapsBuilder {
	if IsNil(format) {
		panic("format cannot be nil")
	}
	b.format = format
	return b
}

func (b *MapsBuilder) NewMaps() *Maps {
	maps := &Maps{
		CallbackBase: b.CallbackBase(),
		source:       b.source,
		target:       b.target,
	}
	if format := b.format; format != nil {
		maps.format = format
		(&Format{as: format}).Require(maps.Metadata())
	}
	return maps
}

func Map[T any](
	handler Handler,
	source  any,
	format  ... any,
) (t T, tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	var builder MapsBuilder
	builder.FromSource(source).
		    ToTarget(&t)
	if len(format) == 1 {
		builder.WithFormat(format[0])
	}
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = NewNotHandledError(maps)
	} else {
		_, tp, err = CoerceResult[T](maps, &t)
	}
	return
}

func MapInto[T any](
	handler Handler,
	source  any,
	target  *T,
	format  ... any,
) (tp *promise.Promise[T], err error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	var builder MapsBuilder
	builder.FromSource(source).
		ToTarget(target)
	if len(format) == 1 {
		builder.WithFormat(format[0])
	}
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = NewNotHandledError(maps)
	} else {
		_, tp, err = CoerceResult[T](maps, target)
	}
	return
}

func MapAll[T any](
	handler Handler,
	source  any,
	format  ... any,
) (t []T, _ *promise.Promise[[]T], _ error) {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if IsNil(source) || reflect.TypeOf(source).Kind() != reflect.Slice {
		panic("source must be a non-nil slice")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	ts := reflect.ValueOf(source)
	t   = make([]T, ts.Len())
	var promises []*promise.Promise[T]
	for i := 0; i < ts.Len(); i++ {
		var builder MapsBuilder
		builder.FromSource(ts.Index(i).Interface()).ToTarget(&t[i])
		if len(format) == 1 {
			builder.WithFormat(format[0])
		}
		maps := builder.NewMaps()
		if result := handler.Handle(maps, false, nil); result.IsError() {
			return nil, nil, result.Error()
		} else if !result.handled {
			return nil, nil, NewNotHandledError(maps)
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
	ErrFormatMissing   = errors.New("the Format constraint requires a non-empty `as:format` tag")
)