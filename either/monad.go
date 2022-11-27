package either

import "fmt"

type (
	// Either represents one of two values (left or right).
	Either[L, R any] interface{}

	// right represents the right side of an Either.
	right[R any] struct {
		val R
	}

	// left represents the left side of an Either.
	left[L any] struct {
		val L
	}
)

// Left returns a new Either with a left value.
func Left[L any](val L) left[L] {
	return left[L]{val}
}

// Right returns a new Either with a right value.
func Right[R any](val R) right[R] {
	return right[R]{val}
}

// Seq (seq)
func Seq[L, R, R2 any](_ Either[L, R], e Either[L, R2]) Either[L, R2] {
	return e
}

// Map (map/fmap)
func Map[L, R, R2 any](e Either[L, R], f func(R) R2) Either[L, R2] {
	if e == nil {
		panic("e cannot be nil")
	}
	if f == nil {
		panic("fun cannot be nil")
	}
	if r, ok := e.(right[R]); ok {
		return right[R2]{f(r.val)}
	}
	return e
}

// Apply (apply/<*>/ap)
func Apply[L, R, R2 any](e Either[L, func(R) R2], f Either[L, R]) Either[L, R2] {
	if e == nil {
		panic("e cannot be nil")
	}
	if f == nil {
		panic("f cannot be nil")
	}
	if r, ok := e.(right[func(R) R2]); ok {
		return Map(f, r.val)
	}
	return e
}

// FlatMap (flatMap/bind/chain/liftM)
func FlatMap[L, R, R2 any](e Either[L, R], f func(R) Either[L, R2]) Either[L, R2] {
	if e == nil {
		panic("e cannot be nil")
	}
	if f == nil {
		panic("f cannot be nil")
	}
	if r, ok := e.(right[R]); ok {
		return f(r.val)
	}
	return e
}

// MapLeft (mapLeft)
func MapLeft[L, L2, R any](e Either[L, R], f func(L) L2) Either[L2, R] {
	if e == nil {
		panic("e cannot be nil")
	}
	if f == nil {
		panic("f cannot be nil")
	}
	if l, ok := e.(left[L]); ok {
		return left[L2]{f(l.val)}
	}
	return e
}

// Fold (fold/either)
func Fold[L, R, A any](e Either[L, R], l func(L) A, r func(R) A) A {
	if e == nil {
		panic("e cannot be nil")
	}
	var a A
	switch v := e.(type) {
	case left[L]:
		if l != nil {
			a = l(v.val)
		}
	case right[R]:
		if r != nil {
			a = r(v.val)
		}
	default:
		panic(fmt.Sprintf("invalid either: %+v", e))
	}
	return a
}

// Match (fold/either)
func Match[L, R any](e Either[L, R], l func(L), r func(R)) {
	if e == nil {
		panic("e cannot be nil")
	}
	switch v := e.(type) {
	case left[L]:
		if l != nil {
			l(v.val)
		}
	case right[R]:
		if r != nil {
			r(v.val)
		}
	default:
		panic(fmt.Sprintf("invalid either: %+v", e))
	}
}