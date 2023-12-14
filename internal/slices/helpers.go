package slices

import (
	"fmt"
)

// Contains checks for the existence of v in s.
func Contains[E comparable](s []E, v E) bool {
	for _, s := range s {
		if v == s {
			return true
		}
	}
	return false
}

type MapFunc[IN, OUT any] interface {
	~func(int, IN) OUT | ~func(IN) OUT
}

// Map turns a []IN to a []OUT using a mapping function.
// This function has two type parameters, IN and OUT.
// This works with slices of any type.
func Map[IN, OUT any, F MapFunc[IN, OUT]](in []IN, fun F) []OUT {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		var out []OUT
		return out
	}
	f := func(i int, item IN) OUT {
		switch typ := any(fun).(type) {
		case func(int, IN) OUT:
			return typ(i, item)
		case func(IN) OUT:
			return typ(item)
		}
		panic(fmt.Sprintf("unrecognized Map function type %T", fun))
	}
	out := make([]OUT, len(in))
	for i, item := range in {
		out[i] = f(i, item)
	}
	return out
}

type FlatMapFunc[IN, OUT any] interface {
	~func(int, IN) []OUT | ~func(IN) []OUT
}

// FlatMap turns a []IN to a []OUT using a mapping function.
// This function has two type parameters, IN and OUT.
// This works with slices of any type.
func FlatMap[IN, OUT any, F FlatMapFunc[IN, OUT]](in []IN, fun F) []OUT {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		var out []OUT
		return out
	}
	f := func(i int, item IN) []OUT {
		switch typ := any(fun).(type) {
		case func(int, IN) []OUT:
			return typ(i, item)
		case func(IN) []OUT:
			return typ(item)
		}
		panic(fmt.Sprintf("unrecognized Map function type %T", fun))
	}
	var out []OUT
	for i, item := range in {
		out = append(out, f(i, item)...)
	}
	return out
}

type FilterFunc[IN any] interface {
	~func(int, IN) bool | ~func(IN) bool
}

// Filter filters values from a slice using a filter function.
// Build returns a new slice with only the elements of s
// for which f returned true.
func Filter[IN any, F FilterFunc[IN]](in []IN, fun F) []IN {
	var out []IN
	if len(in) == 0 {
		return in
	}
	f := func(i int, item IN) bool {
		switch typ := any(fun).(type) {
		case func(int, IN) bool:
			return typ(i, item)
		case func(IN) bool:
			return typ(item)
		default:
			panic(fmt.Sprintf("unrecognized Filter function type %T", fun))
		}
	}
	for i, item := range in {
		if f(i, item) {
			out = append(out, item)
		}
	}
	return out
}

// OfType filters values from a slice satisfying a given type.
func OfType[IN, T any](in []IN) []T {
	var out []T
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		var a any = item
		if tt, ok := a.(T); ok {
			out = append(out, tt)
		}
	}
	return out
}

type AccumulatorFunc[IN, OUT any] func(out OUT, i int, item IN) OUT

// Reduce reduces a []IN to a single value using an accumulator function.
func Reduce[IN, OUT any](
	in []IN, initializer OUT,
	fun AccumulatorFunc[IN, OUT],
) OUT {
	out := initializer
	for i, item := range in {
		out = fun(out, i, item)
	}
	return out
}

// Remove removes all items from the slice in place.
func Remove[IN comparable](in []IN, items ...IN) []IN {
	for _, item := range items {
		if len(in) == 0 {
			return in
		}
		for ii, s := range in {
			if s == item {
				in[ii] = in[len(in)-1]
				in = in[:len(in)-1]
				break
			}
		}
	}
	return in
}

// First returns the First element (or zero value if empty) and bool if exists.
func First[IN any](in []IN) (IN, bool) {
	if len(in) > 0 {
		return in[0], true
	}
	var zero IN
	return zero, false
}

// Last returns the Last element (or zero value if empty) and bool if exists.
func Last[IN any](in []IN) (IN, bool) {
	if len(in) > 0 {
		return in[len(in)-1], true
	}
	var zero IN
	return zero, false
}
