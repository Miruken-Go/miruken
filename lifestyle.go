package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"math"
	"sync"
)

type (
	// LifestyleProvider is a FilterProvider of lifestyles.
	LifestyleProvider struct {
		filters []Filter
	}

	// Lifestyle provides common lifestyle functionality.
	Lifestyle struct{}

	// LifestyleEntry stores a lazy Instance.
	LifestyleEntry struct {
		Instance []any
		Once     *sync.Once
	}

	// LifestyleCache maintains a cache of LifestyleEntry's.
	LifestyleCache map[any]*LifestyleEntry
)


func (l *Lifestyle) Order() int {
	return math.MaxInt32 - 1000
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
	binding Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return l.filters, nil
}

func (l *LifestyleProvider) SetFilters(filters ...Filter) {
	if len(filters) == 0 {
		panic("filters cannot be empty")
	}
	if l.filters != nil {
		panic("lifestyle can only be set Once")
	}
	l.filters = filters
}


// Singleton

type (
	// Singleton LifestyleProvider providing same Instance.
	Singleton struct {
		LifestyleProvider
	}

	// singleton is a Filter that caches an Instance.
	singleton struct {
		Lifestyle
		keys LifestyleCache
		lock sync.RWMutex
	}
)


func (s *Singleton) Init() error {
	s.SetFilters(&singleton{})
	return nil
}

func (s *singleton) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	key := ctx.Callback().(*Provides).Key()

	var entry *LifestyleEntry
	s.lock.RLock()
	if keys := s.keys; keys != nil {
		entry = keys[key]
	}
	s.lock.RUnlock()

	if entry == nil {
		s.lock.Lock()
		if keys := s.keys; keys != nil {
			if entry = keys[key]; entry == nil {
				entry     = &LifestyleEntry{Once: new(sync.Once)}
				keys[key] = entry
			}
		} else {
			entry  = &LifestyleEntry{Once: new(sync.Once)}
			s.keys = LifestyleCache{key: entry}
		}
		s.lock.Unlock()
	}

	entry.Once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("singleton: panic: %v", r)
				}
				entry.Once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			entry.Once = new(sync.Once)
		} else {
			entry.Instance = out
		}
	})

	return entry.Instance, nil, err
}

