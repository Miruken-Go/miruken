package koanf

import (
	"fmt"
	"github.com/knadh/koanf"
	"github.com/miruken-go/miruken"
	"reflect"
)

// factory of configurations populated by the koanf library.
// https://github.com/knadh/koanf
type factory struct {
	k *koanf.Koanf
}

// NewConfiguration return a new instance populated from the
// sources supplied to the shared Koanf instance.
func (f *factory) NewConfiguration(
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
		if err := f.k.Unmarshal("", out); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if ptr {
			out = reflect.ValueOf(out).Elem().Interface()
		}
		return out, nil
	}
	return nil, nil
}


// Use returns a factory using the shared Koanf instance.
func Use(k *koanf.Koanf) any {
	if k == nil {
		panic("k cannot be nil")
	}
	return &factory{k}
}
