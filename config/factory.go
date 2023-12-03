package config

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/provides"
	"maps"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

type (
	// Factory of configurations using assigned Provider.
	Factory struct {
		Provider
		lock  sync.Mutex
		cache atomic.Pointer[map[loadKey]any]
	}

	// Load restricts resolutions to configurations only.
	Load struct {
		Path string
		Flat bool
	}

	loadKey struct{
		typ  reflect.Type
		path string
		flat bool
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
// populated from the designated Provider.
func (f *Factory) NewConfiguration(
	_*struct{args.Strict; Load}, p *provides.It,
) (any, error) {
	if typ, ok := p.Key().(reflect.Type); ok {
		var path string
		var flat bool
		if load, ok := constraints.First[*Load](p); ok {
			path = load.Path
			flat = load.Flat
		}

		// Check cache first
		key := loadKey{typ: typ, path: path, flat: flat}
		if cache := f.cache.Load(); cache != nil {
			if o, ok := (*cache)[key]; ok {
				return o, nil
			}
		}

		// Use copy-on-write idiom since reads should be more frequent than writes.
		f.lock.Lock()
		defer f.lock.Unlock()

		var cc map[loadKey]any
		cache := f.cache.Load()
		if cache != nil {
			if o, ok := (*cache)[key]; ok {
				return o, nil
			}
			cc = maps.Clone(*cache)
		} else {
			cc = make(map[loadKey]any, 1)
		}

		var out any
		ptr := typ.Kind() == reflect.Ptr
		if ptr {
			out = reflect.New(typ.Elem()).Interface()
		} else {
			out = reflect.New(typ).Interface()
		}
		if err := f.Unmarshal(path, flat, out); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		if !ptr {
			out = reflect.ValueOf(out).Elem().Interface()
		}

		if v, ok := out.(interface {
			Validate() error
		}); ok {
			if err := v.Validate(); err != nil {
				return nil, fmt.Errorf("config: %w", err)
			}
		}

		cc[key] = out
		f.cache.Store(&cc)
		return out, nil
	}
	return nil, nil
}
