package slices

import (
	"fmt"
)

type MapFunc[T1, T2 any] interface {
	~func(int, T1) T2 | ~func(T1) T2
}

// Map turns a []T1 to a []T2 using a mapping function.
// This function has two type parameters, T1 and T2.
// This works with slices of any type.
func Map[T1, T2 any, F MapFunc[T1, T2]](s []T1, fun F) []T2 {
	if s == nil {
		return nil
	}
	if len(s) == 0 {
		var r []T2
		return r
	}
	f := func(i int, ss T1) T2 {
		switch tf := (any)(fun).(type) {
		case func(int, T1) T2:
			return tf(i, ss)
		case func(T1) T2:
			return tf(ss)
		}
		panic(fmt.Sprintf("unrecognized Map function type %T", fun))
	}
	r := make([]T2, len(s))
	for i, t := range s {
		r[i] = f(i, t)
	}
	return r
}

type FilterFunc[T any] interface {
	~func(int, T) bool | ~func(T) bool
}

// Filter filters values from a slice using a filter function.
// It returns a new slice with only the elements of s
// for which f returned true.
func Filter[T any, F FilterFunc[T]](s []T, fun F) []T {
	var r []T
	if len(s) == 0 {
		return s
	}
	f := func(i int, s T) bool {
		switch tf := (any)(fun).(type) {
		case func(int, T) bool:
			return tf(i, s)
		case func(T) bool:
			return tf(s)
		default:
			panic(fmt.Sprintf("unrecognized Filter function type %T", fun))
		}
	}
	for i, t := range s {
		if f(i, t) {
			r = append(r, t)
		}
	}
	return r
}

// OfType filters values from a slice satisfying a given type.
func OfType[T1, T2 any](s []T1) []T2 {
	var r []T2
	if len(s) == 0 {
		return r
	}
	for _, t := range s {
		var a any = t
		if tt, ok := a.(T2); ok{
			r = append(r, tt)
		}
	}
	return r
}

type AccumulatorFunc[T1, T2 any] func(r T2, i int, s T1) T2

// Reduce reduces a []T1 to a single value using an accumulator function.
func Reduce[T1, T2 any, F AccumulatorFunc[T1, T2]](
	s []T1, initializer T2,
	f AccumulatorFunc[T1, T2],
) T2 {
	r := initializer
	if s != nil {
		for i, t := range s {
			r = f(r, i, t)
		}
	}
	return r
}

// First returns the First element (or zero value if empty) and bool if exists.
func First[T any](s []T) (T, bool) {
	if len(s) > 0 {
		return s[0], true
	}
	var zero T
	return zero, false
}

// Last returns the Last element (or zero value if empty) and bool if exists.
func Last[T any](s []T) (T, bool) {
	if len(s) > 0 {
		return s[len(s)-1], true
	}
	var zero T
	return zero, false
}

