package miruken

import (
	"errors"
	"reflect"
	"strings"
)

// Maps callbacks bivariantly.
type Maps struct {
	CallbackBase
	source   any
	target   any
	format   any
	metadata BindingMetadata
}

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

func (m *Maps) Metadata() *BindingMetadata {
	return &m.metadata
}

func (m *Maps) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, m, greedy, composer)
}

type Format struct {
	as any
}

func (f *Format) InitWithTag(tag reflect.StructTag) error {
	if as, ok := tag.Lookup("as"); ok {
		if format := strings.TrimSpace(as); len(format) > 0 {
			f.as = format
		}
	}
	if IsNil(f.as){
		return errors.New("the Format constraint requires a non-empty `as:format` tag")
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

// MapsBuilder builds Maps callbacks.
type MapsBuilder struct {
	CallbackBuilder
	source any
	target any
	format any
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
		maps.format   = format
		maps.metadata = BindingMetadata{}
		(&Format{as: format}).Require(&maps.metadata)
	}
	return maps
}

func Map[T any](
	handler Handler,
	source  any,
	format  ... any,
) (T, error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	var target T
	var builder MapsBuilder
	builder.FromSource(source).
		    ToTarget(&target)
	if len(format) == 1 {
		builder.WithFormat(format[0])
	}
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		return target, result.Error()
	} else if !result.handled {
		return target, NotHandledError{maps}
	}
	maps.CopyResult(TargetValue(&target), false)
	return target, nil
}

func MapInto[T any](
	handler Handler,
	source  any,
	target  *T,
	format  ... any,
) error {
	if handler == nil {
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
		return result.Error()
	} else if !result.handled {
		return NotHandledError{maps}
	}
	maps.CopyResult(TargetValue(target), false)
	return nil
}

func MapAll[T any](
	handler Handler,
	source  any,
	format  ... any,
) ([]T, error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if IsNil(source) || reflect.TypeOf(source).Kind() != reflect.Slice {
		panic("source must be a non-nil slice")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	var target []T
	ts      := reflect.ValueOf(source)
	tv      := TargetSliceValue(&target)
	tt      := tv.Type().Elem().Elem()
	te      := reflect.New(tt).Interface()
	results := make([]any, ts.Len())
	for i := 0; i < ts.Len(); i++ {
		var builder MapsBuilder
		builder.FromSource(ts.Index(i).Interface()).ToTarget(te)
		if len(format) == 1 {
			builder.WithFormat(format[0])
		}
		maps := builder.NewMaps()
		if result := handler.Handle(maps, false, nil); result.IsError() {
			return target, result.Error()
		} else if !result.handled {
			return target, NotHandledError{maps}
		}
		results[i] = maps.Result(false)
	}
	CopySliceIndirect(results, &target)
	return target, nil
}

var (
	_mapsPolicy Policy = &BivariantPolicy{}
	_formatType        = TypeOf[*Format]()
)