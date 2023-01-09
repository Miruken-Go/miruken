package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"math"
	"sync"
)

// LifestyleProvider is a FilterProvider of lifestyles.
type LifestyleProvider struct {
	filters []Filter
}

func (l *LifestyleProvider) Required() bool {
	return true
}

func (l *LifestyleProvider) AppliesTo(
	callback Callback,
) bool {
	_, ok := callback.(*Provides)
	return ok
}

func (l *LifestyleProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return l.filters, nil
}

func (l *LifestyleProvider) SetFilters(filters ... Filter) {
	if len(filters) == 0 {
		panic("filters cannot be empty")
	}
	if l.filters != nil {
		panic("lifestyle can only be set once")
	}
	l.filters = filters
}

// Lifestyle provides common lifestyle functionality.
type Lifestyle struct{}

func (l *Lifestyle) Order() int {
	return math.MaxInt32 - 1000
}

// Singleton LifestyleProvider providing same instance.
type Singleton struct {
	LifestyleProvider
}

func (s *Singleton) Init() error {
	s.SetFilters(&singleton{})
	return nil
}

type (
	// lifestyleEntry stores a lazy instance.
	lifestyleEntry struct {
		instance []any
		once     *sync.Once
	}

	// lifestyleCache maintains a cache of lifestyleEntry's.
	lifestyleCache map[any]*lifestyleEntry
)

// singleton is a Filter that caches an instance.
type singleton struct {
	Lifestyle
	keys lifestyleCache
	lock sync.RWMutex
}

func (s *singleton) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	key := ctx.Callback().(*Provides).Key()

	var entry *lifestyleEntry
	s.lock.RLock()
	if keys := s.keys; keys != nil {
		entry = keys[key]
	}
	s.lock.RUnlock()

	if entry == nil {
		s.lock.Lock()
		if keys := s.keys; keys != nil {
			if entry = keys[key]; entry == nil {
				entry     = &lifestyleEntry{once: new(sync.Once)}
				keys[key] = entry
			}
		} else {
			entry  = &lifestyleEntry{once: new(sync.Once)}
			s.keys = lifestyleCache{key: entry}
		}
		s.lock.Unlock()
	}

	entry.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("singleton: panic: %v", r)
				}
				entry.once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			entry.once = new(sync.Once)
		} else {
			entry.instance = out
		}
	})

	return entry.instance, nil, err
}
