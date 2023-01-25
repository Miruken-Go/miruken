package config

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/slices"
	"reflect"
	"strings"
)

type (
	// Factory of configurations using assigned Provider.
	Factory struct {
		provider Provider
	}

	// Load restricts miruken.Provides to configurations.
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
		miruken.Provides
		miruken.Singleton
		Load
	  }, provides *miruken.Provides,
) (any, error) {
	if typ, ok := provides.Key().(reflect.Type); ok {
		var out any
		ptr := typ.Kind() == reflect.Ptr
		if ptr {
			out = reflect.New(typ.Elem()).Interface()
		} else {
			out = reflect.New(typ).Interface()
		}
		load := loadPreference(provides)
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

func loadPreference(provides *miruken.Provides) *Load {
	for _, c := range provides.Constraints() {
		if l, ok := c.(*Load); ok {
			return l
		}
	}
	return &Load{}
}
