package config

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/provides"
	"reflect"
	"strings"
)

type (
	// Factory of configurations using assigned Provider.
	Factory struct {
		Provider
	}

	// Load restricts resolutions to configurations only.
	Load struct {
		Path string
		Flat bool
	}
)


// Load

func (l *Load) Required() bool {
	return true
}

func (l *Load) Implied() bool {
	return false
}

func (l *Load) InitWithTag(tag reflect.StructTag) error {
	if path, ok := tag.Lookup("path"); ok && len(strings.TrimSpace(path)) > 0 {
		parts := strings.Split(path, ",")
		l.Path = parts[0]
		l.Flat = slices.Contains(parts, "flat")
	}
	return nil
}

func (l *Load) Satisfies(required miruken.Constraint, _ miruken.Callback) bool {
	_, ok := required.(*Load)
	return ok
}


// Factory

// NoConstructor prevents Factory from being created implicitly.
// The Factory is explicitly created by the Installer which
// assigns the Provider.
func (f *Factory) NoConstructor() {}

// NewConfiguration return a new configuration instance
// populated by the assigned Provider.
func (f *Factory) NewConfiguration(
	_*struct{
		provides.It; args.Strict; Load
	}, p *provides.It,
) (any, error) {
	if typ, ok := p.Key().(reflect.Type); ok {
		var out any
		ptr := typ.Kind() == reflect.Ptr
		if ptr {
			out = reflect.New(typ.Elem()).Interface()
		} else {
			out = reflect.New(typ).Interface()
		}
		var path string
		var flat bool
		if load, ok := constraints.First[*Load](p); ok {
			path = load.Path
			flat = load.Flat
		}
		if err := f.Unmarshal(path, flat, out); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if !ptr {
			out = reflect.ValueOf(out).Elem().Interface()
		}
		return out, nil
	}
	return nil, nil
}

