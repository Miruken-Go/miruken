package slices

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Safe is a slice wrapper  to provide some concurrent operations.
// It is optimized for reads using the copy-on-write idiom.
type Safe[T any] struct {
	items atomic.Pointer[[]T]
	lock sync.Mutex
}


// NewSafe create a Safe slice with initial items.
func NewSafe[T any](initial ...T) *Safe[T] {
	return (&Safe[T]{}).Reset(initial...)
}


func (s *Safe[T]) Items() []T {
	if items := s.items.Load(); items != nil {
		return *items
	}
	return nil
}

func (s *Safe[T]) Reset(items ...T) *Safe[T] {
	c := append([]T{}, items...)
	s.items.Store(&c)
	return s
}

func (s *Safe[T]) Index(eq func(T) bool) int {
	if eq == nil {
		panic("eq func cannot be nil")
	}
	for i, v := range s.Items() {
		if eq(v) {
			return i
		}
	}
	return -1
}

func (s *Safe[T]) Append(items ...T) *Safe[T] {
	if len(items) > 0 {
		s.lock.Lock()
		defer s.lock.Unlock()
		s1 := s.Items()
		s2 := make([]T, len(s1)+len(items))
		copy(s2, s1)
		copy(s2[len(s1):], items)
		s.items.Store(&s2)
	}
	return s
}

func (s *Safe[T]) Insert(index int, items ...T) *Safe[T] {
	if index < 0 {
		panic("index must be >= 0")
	}
	if len(items) > 0 {
		if index > len(items) {
			panic(fmt.Sprintf("index must be <= %v", len(items)))
		}
		s.lock.Lock()
		defer s.lock.Unlock()
		s1 := s.Items()
		s2 := make([]T, len(s1)+len(items))
		copy(s2, (s1)[:index])
		copy(s2[index:], items)
		copy(s2[index+len(items):], (s1)[index:])
		s.items.Store(&s2)
	}
	return s
}

func (s *Safe[T]) Delete(eq func(T) (bool, bool)) *Safe[T] {
	if eq == nil {
		panic("eq func cannot be nil")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	var s2 []T
	for _, item := range s.Items() {
		match, stop := eq(item)
		if !match {
			s2 = append(s2, item)
		}
		if stop {
			break
		}
	}
	s.items.Store(&s2)
	return s
}

// Item create a function that check equality
// to a supplied literal.
// It requires comparable items.
func Item[T comparable](t T) func(T) bool {
	return func(item T) bool {
		return item == t
	}
}