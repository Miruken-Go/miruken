package miruken

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"regexp"
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

	// FormatRule describes how to interpret the format.
	FormatRule uint8

	// Format is a BindingConstraint for applying formatting.
	Format struct {
		direction  FormatDirection
		rule       FormatRule
		identifier string
		pattern    *regexp.Regexp
	}
)

const (
	FormatDirectionNone FormatDirection = 0
	FormatDirectionTo FormatDirection = 1 << iota
	FormatDirectionFrom

	FormatRuleEquals     FormatRule = 0
	FormatRuleStartsWith FormatRule = 1 << iota
	FormatRuleEndsWith
	FormatRulePattern
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

func (f *Format) Direction() FormatDirection {
	return f.direction
}

func (f *Format) Rule() FormatRule {
	return f.rule
}

func (f *Format) Identifier() string {
	return f.identifier
}

func (f *Format) InitWithTag(tag reflect.StructTag) error {
	if to, ok := tag.Lookup("to"); ok {
		f.direction = FormatDirectionTo
		return f.parse(to)
	} else if from, ok := tag.Lookup("from"); ok {
		f.direction = FormatDirectionFrom
		return f.parse(from)
	}
	return ErrInvalidFormat
}

func (f *Format) Merge(constraint BindingConstraint) bool {
	if format, ok := constraint.(*Format); ok {
		*f = *format
		return true
	}
	return false
}

func (f *Format) Require(metadata *BindingMetadata) {
	if identifier := f.identifier; len(identifier) > 0 {
		metadata.Set(_formatType, f)
	}
}

func (f *Format) Matches(metadata *BindingMetadata) bool {
	if m, ok := metadata.Get(_formatType); ok {
		if format, ok := m.(*Format); ok {
			if f.direction != format.direction {
				return false
			}
			switch f.rule {
			case FormatRuleEquals:
				switch format.rule {
				case FormatRuleEquals:
					return f.identifier == format.identifier
				case FormatRuleStartsWith:
					return strings.HasPrefix(f.identifier, format.identifier)
				case FormatRuleEndsWith:
					return strings.HasSuffix(f.identifier, format.identifier)
				case FormatRulePattern:
					return format.pattern.MatchString(f.identifier)
				}
			case FormatRuleStartsWith:
				switch format.rule {
				case FormatRuleEquals, FormatRuleStartsWith:
					return strings.HasPrefix(format.identifier, f.identifier)
				case FormatRulePattern:
					return format.pattern.MatchString(f.identifier)
				}
			case FormatRuleEndsWith:
				switch format.rule {
				case FormatRuleEquals:
					return strings.HasSuffix(format.identifier, f.identifier)
				case FormatRuleEndsWith:
					return strings.HasSuffix(f.identifier, format.identifier)
				case FormatRulePattern:
					return format.pattern.MatchString(f.identifier)
				}
			case FormatRulePattern:
				switch format.rule {
				case FormatRuleEquals, FormatRuleStartsWith, FormatRuleEndsWith:
					return f.pattern.MatchString(format.identifier)
				}
			}
		}
	}
	return false
}

func (f *Format) parse(format string) error {
	format = strings.TrimSpace(format)
	var start, end int
	var startsWith, endsWith bool
	if strings.HasPrefix(format, "//") {
		start = 1
	} else if strings.HasPrefix(format, "/") {
		start      = 1
		startsWith = true
	}
	if strings.HasSuffix(format, "//") {
		end = 1
	} else if strings.HasSuffix(format, "/") {
		end      = 1
		endsWith = true
	}
	if start > 0 || end > 0 {
		format = strings.TrimSpace(format[start:len(format)-end])
	}
	if len(format) == 0 {
		return ErrEmptyFormatIdentifier
	}
	if startsWith {
		if endsWith {
			if regex, err := regexp.Compile(format); err != nil {
				return fmt.Errorf("invalid format pattern: %w", err)
			} else {
				f.pattern = regex
			}
			f.rule = FormatRulePattern
		} else {
			f.rule = FormatRuleStartsWith
		}
	} else if endsWith {
		f.rule = FormatRuleEndsWith
	}
	f.identifier = format
	return nil
}

// To maps to a format.
func To(format string) *Format {
	f := &Format{direction: FormatDirectionTo}
	if err := f.parse(format); err != nil {
		panic(err)
	}
	return f
}

// From maps from a format.
func From(format string) *Format {
	f := &Format{direction: FormatDirectionFrom}
	if err := f.parse(format); err != nil {
		panic(err)
	}
	return f
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
	_mapsPolicy Policy       = &BivariantPolicy{}
	_formatType              = TypeOf[*Format]()
	ErrInvalidFormat         = errors.New("invalid format tag")
	ErrEmptyFormatIdentifier = errors.New("empty format identifier")
)