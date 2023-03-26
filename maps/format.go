package maps

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"reflect"
	"regexp"
	"strings"
)

type (
	// Direction indicates direction of formatting.
	Direction uint8

	// FormatRule describes how to interpret the format.
	FormatRule uint8

	// Format is a Constraint for applying formatting.
	Format struct {
		name      string
		direction Direction
		rule      FormatRule
		pattern   *regexp.Regexp
		params    map[string]string
	}
)


const (
	DirectionNone Direction = 0
	DirectionTo   Direction = 1 << iota
	DirectionFrom

	FormatRuleEquals     FormatRule = 0
	FormatRuleStartsWith FormatRule = 1 << iota
	FormatRuleEndsWith
	FormatRulePattern
	FormatRuleAll
)


func (f *Format) Name() string {
	return f.name
}

func (f *Format) Direction() Direction {
	return f.direction
}

func (f *Format) Rule() FormatRule {
	return f.rule
}

func (f *Format) Params() map[string]string {
	return f.params
}

func (f *Format) Required() bool {
	return true
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

func (f *Format) Merge(constraint miruken.Constraint) bool {
	if format, ok := constraint.(*Format); ok {
		*f = *format
		return true
	}
	return false
}

func (f *Format) Satisfies(required miruken.Constraint) bool {
	rf, ok := required.(*Format)
	if !ok {
		return false
	}
	if f.direction != rf.direction {
		return false
	}
	if f.rule == FormatRuleAll || rf.rule == FormatRuleAll {
		return true
	}
	switch rf.rule {
	case FormatRuleEquals:
		switch f.rule {
		case FormatRuleEquals:
			return rf.name == f.name
		case FormatRuleStartsWith:
			return strings.HasPrefix(rf.name, f.name)
		case FormatRuleEndsWith:
			return strings.HasSuffix(rf.name, f.name)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.name)
		}
	case FormatRuleStartsWith:
		switch f.rule {
		case FormatRuleEquals, FormatRuleStartsWith:
			return strings.HasPrefix(rf.name, f.name)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.name)
		}
	case FormatRuleEndsWith:
		switch f.rule {
		case FormatRuleEquals:
			return strings.HasSuffix(rf.name, f.name)
		case FormatRuleEndsWith:
			return strings.HasSuffix(f.name, rf.name)
		case FormatRulePattern:
			return f.pattern.MatchString(rf.name)
		}
	case FormatRulePattern:
		switch f.rule {
		case FormatRuleEquals, FormatRuleStartsWith, FormatRuleEndsWith:
			return rf.pattern.MatchString(f.name)
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
	if format == "*" {
		f.rule = FormatRuleAll
		f.name = "*"
		return nil
	}
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
	f.name = format
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

var (
	ErrInvalidFormat          = errors.New("invalid format tag")
	ErrEmptyFormatIdentifier  = errors.New("empty format name")
)