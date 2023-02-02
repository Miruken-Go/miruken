package config

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/slices"
	"reflect"
	"strings"
)

type (
	// Factory of configurations using assigned Provider.
	Factory struct {
		provider Provider
	}

	// Load restricts provides.Provides to configurations.
	Load struct {
		Path string
		Flat bool
	}
)


// Load

func (l *Load) Required() bool {
	return true
}

func (l *Load) InitWithTag(tag reflect.StructTag) error {
	if path, ok := tag.Lookup("path"); ok && len(strings.TrimSpace(path)) > 0 {
		parts := strings.Split(path, ",")
		l.Path = parts[0]
		l.Flat = slices.Contains(parts, "flat")
	}
	return nil
}

func (l *Load) Satisfies(required miruken.BindingConstraint) bool {
	_, ok := required.(*Load)
	return ok
}


// Factory

// NewConfiguration return a new configuration instance
// populated by the assigned Provider.
func (f *Factory) NewConfiguration(
	_*struct{
		provides.It
		provides.Singleton
		Load
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
		load := loadPreference(p)
		if err := f.provider.Unmarshal(load.Path, load.Flat, out); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if !ptr {
			out = reflect.ValueOf(out).Elem().Interface()
		}
		return out, nil
	}
	return nil, nil
}

func loadPreference(p *provides.It) *Load {
	for _, c := range p.Constraints() {
		if l, ok := c.(*Load); ok {
			return l
		}
	}
	return &Load{}
}
