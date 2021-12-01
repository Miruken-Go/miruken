package miruken

import (
	"errors"
	"reflect"
	"strings"
)

// Maps callbacks Bivariantly.
type Maps struct {
	CallbackBase
	source   interface{}
	target   interface{}
	format   interface{}
	metadata BindingMetadata
}

func (m *Maps) Source() interface{} {
	return m.source
}

func (m *Maps) Target() interface{} {
	return m.target
}

func (m *Maps) Format() interface{} {
	return m.format
}

func (m *Maps) Key() interface{} {
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

func (m *Maps) ReceiveResult(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler,
) (accepted bool) {
	return m.AddResult(result)
}

func (m *Maps) Dispatch(
	handler  interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, m.source, m, greedy, composer, m).
		OtherwiseHandledIf(m.Result() != nil)
}

type Format struct {
	as map[interface{}]struct{}
}

func (f *Format) InitWithTag(tag reflect.StructTag) error {
	if as, ok := tag.Lookup("as"); ok {
		f.as = make(map[interface{}]struct{})
		for _, format := range strings.Split(as, ",") {
			if format = strings.TrimSpace(format); len(format) > 0 {
				f.as[format] = struct{}{}
			}
		}
	}
	if len(f.as) == 0 {
		return errors.New("the Format constraint requires a non-empty `as:[formats]` tag")
	}
	return nil
}

func (f *Format) Require(metadata *BindingMetadata) {
	if as := f.as; len(as) == 1 {
		for format := range as {
			metadata.Set(reflect.TypeOf(f), format)
		}
	} else {
		panic("Format can only be required for a single format")
	}
}

func (f *Format) Matches(metadata *BindingMetadata) bool {
	var found bool
	if format, ok := metadata.Get(reflect.TypeOf(f)); ok {
		_, found = f.as[format]
	}
	return found
}

// MapsBuilder builds Maps callbacks.
type MapsBuilder struct {
	CallbackBuilder
	source interface{}
	target interface{}
	format interface{}
}

func (b *MapsBuilder) FromSource(
	source interface{},
) *MapsBuilder {
	if IsNil(source) {
		panic("source cannot be nil")
	}
	b.source = source
	return b
}

func (b *MapsBuilder) ToTarget(
	target interface{},
) *MapsBuilder {
	if IsNil(target) {
		panic("target cannot be nil")
	}
	b.target = target
	return b
}

func (b *MapsBuilder) WithFormat(
	format interface{},
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
		(&Format{as: map[interface{}]struct{}{
			format: {},
		}}).Require(&maps.metadata)
	}
	return maps
}

func Map(
	handler Handler,
	source interface{},
	target interface{},
	format ... interface{},
) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	builder := new(MapsBuilder).
		FromSource(source).
		ToTarget(target)
	if len(format) == 1 {
		builder = builder.WithFormat(format[0])
	}
	maps := builder.NewMaps()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		return result.Error()
	} else if !result.handled {
		return NotHandledError{maps}
	}
	tv := TargetValue(target)
	if tv.Elem().Kind() == reflect.Ptr {
		maps.CopyResult(tv)
	}
	return nil
}

func MapAll(
	handler Handler,
	source []interface{},
	target interface{},
	format ... interface{},
) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if len(format) > 1 {
		panic("only one format is allowed")
	}
	tv      := TargetSliceValue(target)
	tt      := tv.Type().Elem().Elem()
	results := make([]interface{}, len(source))
	for i, src := range source {
		builder := new(MapsBuilder).
			FromSource(src).
			ToTarget(tt)
		if len(format) == 1 {
			builder = builder.WithFormat(format[0])
		}
		maps := builder.NewMaps()
		if result := handler.Handle(maps, false, nil); result.IsError() {
			return result.Error()
		} else if !result.handled {
			return NotHandledError{maps}
		}
		results[i] = maps.Result()
	}
	CopySliceIndirect(results, target)
	return nil
}

var _mapsPolicy Policy = &BivariantPolicy{}