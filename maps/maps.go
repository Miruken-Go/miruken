package maps

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"regexp"
	"strings"
)

type (
	// It maps callbacks bivariantly.
	It struct {
		miruken.CallbackBase
		key    any
		source any
		target any
	}

	// Builder builds It callbacks.
	Builder struct {
		miruken.CallbackBuilder
		key    any
		source any
		target any
	}

	// Direction indicates direction of formatting.
	Direction uint8

	// FormatRule describes how to interpret the format.
	FormatRule uint8

	// Format is a BindingConstraint for applying formatting.
	Format struct {
		direction  Direction
		rule       FormatRule
		identifier string
		pattern    *regexp.Regexp
		params     map[string]string
	}

	// Strict alias for mapping
	Strict = miruken.Strict
)

const (
	DirectionNone Direction = 0
	DirectionTo   Direction = 1 << iota
	DirectionFrom

	FormatRuleEquals     FormatRule = 0
	FormatRuleStartsWith FormatRule = 1 << iota
	FormatRuleEndsWith
	FormatRulePattern
)

// It

func (m *It) Source() any {
	return m.source
}

func (m *It) Target() any {
	return m.target
}

func (m *It) Key() any {
	in := m.key
	if in == nil {
		in = reflect.TypeOf(m.source)
	}
	out := reflect.TypeOf(m.target).Elem()
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
	return fmt.Sprintf("It => %+v", m.source)
}


// Format

func (f *Format) Required() bool {
	return true
}

func (f *Format) Direction() Direction {
	return f.direction
}

func (f *Format) Rule() FormatRule {
	return f.rule
}

func (f *Format) Identifier() string {
	return f.identifier
}

func (f *Format) Params() map[string]string {
	return f.params
}

func (f *Format) InitWithTag(tag reflect.StructTag) error {
	if to, ok := tag.Lookup("to"); ok {
		f.direction = DirectionTo
		return f.parse(to)
	} else if from, ok := tag.Lookup("from"); ok {
		f.direction = DirectionFrom
		return f.parse(from)
	}
	return ErrInvalidFormat
}

func (f *Format) Merge(constraint miruken.BindingConstraint) bool {
	if format, ok := constraint.(*Format); ok {
		*f = *format
		return true
	}
	return false
}

func (f *Format) Satisfies(required miruken.BindingConstraint) bool {
	rf, ok := required.(*Format)
	if !ok {
		return false
	}
	if f.direction != rf.direction {
		return false
	}
	switch rf.rule {
	case FormatRuleEquals:
		switch f.rule {
		case FormatRuleEquals:
			return rf.identifier == f.identifier
		case FormatRuleStartsWith:
			return strings.HasPrefix(rf.identifier, f.identifier)
		case FormatRuleEndsWith:
			return strings.HasSuffix(rf.identifier, f.identifier)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.identifier)
		}
	case FormatRuleStartsWith:
		switch f.rule {
		case FormatRuleEquals, FormatRuleStartsWith:
			return strings.HasPrefix(rf.identifier, f.identifier)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.identifier)
		}
	case FormatRuleEndsWith:
		switch f.rule {
		case FormatRuleEquals:
			return strings.HasSuffix(rf.identifier, f.identifier)
		case FormatRuleEndsWith:
			return strings.HasSuffix(f.identifier, rf.identifier)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.identifier)
		}
	case FormatRulePattern:
		switch f.rule {
		case FormatRuleEquals, FormatRuleStartsWith, FormatRuleEndsWith:
			return rf.pattern.MatchString(f.identifier)
		}
	}
	return false
}

func (f *Format) FlipDirection() *Format {
	flip := *f
	if f.direction == DirectionTo {
		flip.direction = DirectionFrom
	} else {
		flip.direction = DirectionTo
	}
	return &flip
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
func To(format string, params map[string]string) *Format {
	f := &Format{direction: DirectionTo, params: params}
	if err := f.parse(format); err != nil {
		panic(err)
	}
	return f
}

// From maps from a format.
func From(format string, params map[string]string) *Format {
	f := &Format{direction: DirectionFrom, params: params}
	if err := f.parse(format); err != nil {
		panic(err)
	}
	return f
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

func (b *Builder) ToTarget(
	target any,
) *Builder {
	if miruken.IsNil(target) {
		panic("source cannot be nil")
	}
	b.target = target
	return b
}

func (b *Builder) New() *It {
	return &It{
		CallbackBase: b.CallbackBase(),
		key:          b.key,
		source:       b.source,
		target:       b.target,
	}
}

func Map[T any](
	handler miruken.Handler,
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
	maps := builder.New()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: maps}
	} else {
		_, tp, err = miruken.CoerceResult[T](maps, &t)
	}
	return
}

func MapInto[T any](
	handler miruken.Handler,
	source      any,
	target      *T,
	constraints ...any,
) (tp *promise.Promise[T], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if target == nil {
		panic("target cannot be nil")
	}
	var builder Builder
	builder.FromSource(source).
			ToTarget(target).
			WithConstraints(constraints...)
	maps := builder.New()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: maps}
	} else {
		_, tp, err = miruken.CoerceResult[T](maps, target)
	}
	return
}

func MapKey[T any](
	handler miruken.Handler,
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
	maps := builder.New()
	if result := handler.Handle(maps, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: maps}
	} else {
		_, tp, err = miruken.CoerceResult[T](maps, &t)
	}
	return
}

func MapAll[T any](
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
		maps := builder.New()
		if result := handler.Handle(maps, false, nil); result.IsError() {
			return nil, nil, result.Error()
		} else if !result.Handled() {
			return nil, nil, &miruken.NotHandledError{Callback: maps}
		}
		if _, pm, err := miruken.CoerceResult[T](maps, &t[i]); err != nil {
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
	mapsPolicy       miruken.Policy = &miruken.BivariantPolicy{}
	ErrInvalidFormat                = errors.New("invalid format tag")
	ErrEmptyFormatIdentifier = errors.New("empty format identifier")
)