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
	s.SetFilters(&singleton{once: new(sync.Once)})
	return nil
}

// singleton is a Filter that caches an instance.
type singleton struct {
	Lifestyle
	instance []any
	once     *sync.Once
}

func (s *singleton) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	s.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("singleton: panic: %v", r)
				}
				s.once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			s.once = new(sync.Once)
		} else {
			s.instance = out
		}
	})
	return s.instance, nil, err
}
