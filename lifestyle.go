package miruken

import (
	"fmt"
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
	callback interface{},
) bool {
	_, ok := callback.(*Inquiry)
	return ok
}

func (l *LifestyleProvider) Filters(
	binding  Binding,
	callback interface{},
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
	instance []interface{}
	once     *sync.Once
}

func (s *singleton) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  (result []interface{}, err error) {
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
		s.instance, err = next.Filter()
		if err != nil || len(s.instance) == 0 {
			s.once = new(sync.Once)
		}
	})
	return s.instance, err
}
