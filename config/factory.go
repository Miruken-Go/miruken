package config

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"reflect"
)

type (
	// Factory of configurations using assigned Provider.
	Factory struct {
		provider Provider
	}

	// FromConfig is used to request a Configuration value.
	FromConfig struct {
		miruken.Qualifier[FromConfig]
	}
)

// NewConfiguration return a new configuration instance
// populated by the assigned Provider.
func (f *Factory) NewConfiguration(
	_*struct{
		miruken.Provides
		miruken.Singleton
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
		if err := f.provider.Unmarshal("", out); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if ptr {
			out = reflect.ValueOf(out).Elem().Interface()
		}
		return out, nil
	}
	return nil, nil
}