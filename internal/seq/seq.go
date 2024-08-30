package seq

import "iter"

// Empty returns an empty sequence.
func Empty[V any]() iter.Seq[V] {
	return func(yield func(V) bool) {}
}

// Filter selects values from s using a filter func.
func Filter[V any](s iter.Seq[V], f func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range s {
			if f(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Filter2 selects values from s using a filter func with index.
func Filter2[V any](s iter.Seq2[int,V], f func(int, V) bool) iter.Seq2[int,V] {
	return func(yield func(int, V) bool) {
		i := 0
		for ii, v := range s {
			if f(ii, v) {
				if !yield(i, v) {
					return
				}
				i++
			}
		}
	}
}

// OfType selects values from s matching a type.
func OfType[V, U any](s iter.Seq[V]) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range s {
			var a any = v
			if u, ok := a.(U); ok {
				if !yield(u) {
					return
				}
			}
		}
	}
}

// OfType2 selects values from s matching a type with index.
func OfType2[V, U any](s iter.Seq[V]) iter.Seq2[int,U] {
	return func(yield func(int,U) bool) {
		i := 0
		for v := range s {
			var a any = v
			if u, ok := a.(U); ok {
				if !yield(i, u) {
					return
				}
				i++
			}
		}
	}
}

// Map converts values from s using a mapper func.
func Map[V, U any](s iter.Seq[V], f func(V) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Map2 converts values from s using a mapper func with index.
func Map2[V, U any](s iter.Seq2[int,V], f func(int,V) U) iter.Seq2[int,U] {
	return func(yield func(int,U) bool) {
		for i, v := range s {
			if !yield(i, f(i,v)) {
				return
			}
		}
	}
}

// FlatMap converts values from s to a slice using a mapper func.
func FlatMap[V, U any, Slice ~[]U](s iter.Seq[V], f func(V) Slice) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range s {
			for _, u := range f(v) {
				if !yield(u) {
					return
				}
			}
		}
	}
}

// FlatMap2 converts values from s to a slice using a mapper func with index.
func FlatMap2[V, U any, Slice ~[]U](s iter.Seq2[int,V], f func(int,V) Slice) iter.Seq2[int,U] {
	return func(yield func(int,U) bool) {
		i := 0
		for ii, v := range s {
			for _, u := range f(ii, v) {
				if !yield(i, u) {
					return
				}
				i++
			}
		}
	}
}

// Except returns a new sequence with values from s that are not in e.
func Except[V comparable](s, e iter.Seq[V]) iter.Seq[V] {
	exclude := make(map[V]struct{})
	for ev := range e {
		exclude[ev] = struct{}{}
	}
	return Filter(s, func(v V) bool {
		_, ok := exclude[v]
		return !ok
	})
}

// First returns the first value from s.
func First[V any](s iter.Seq[V]) (V, bool) {
	for v := range s {
		return v, true
	}
	var zero V
	return zero, false
}

// Last returns the last value from s.
func Last[V any](s iter.Seq[V]) (v V, ok bool) {
	for v = range s {
		ok = true
	}
	return v, ok
}

// Reduce applies a function to each value in s, accumulating the result.
func Reduce[V, U any](s iter.Seq[V], f func(U,V) U, init U) U {
	for v := range s {
		init = f(init, v)
	}
	return init
}
