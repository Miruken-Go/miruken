package miruken

import (
	"fmt"
	"math"
	"sync"
)

// LifestyleProvider is a FilterProvider for lifestyles.
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

func (l *LifestyleProvider) SetFilter(lifestyle Filter) {
	if lifestyle == nil {
		panic("lifestyle cannot be nil")
	}
	if l.filters != nil {
		panic("lifestyle can only be set once")
	}
	l.filters = []Filter{lifestyle}
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
	s.SetFilter(&singleton{once: new(sync.Once)})
	return nil
}

// singleton is the Filter that captures the instance.
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
