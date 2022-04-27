package miruken

import (
	"fmt"
	"reflect"
)

type (
	// Predicate represents a generic selector.
	Predicate[T any] func(T) bool
)

func combinePredicate2[T any](
	predicate1, predicate2 Predicate[T],
) Predicate[T] {
	if predicate1 == nil {
		return predicate2
	} else if predicate2 == nil {
		return predicate1
	}
	return func(val T) bool {
		if predicate2(val) {
			return true
		}
		if predicate1(val) {
			return true
		}
		return false
	}
}

func CombinePredicates[T any](
	predicate Predicate[T],
	predicates ... Predicate[T],
) Predicate[T] {
	switch len(predicates) {
	case 0: return predicate
	case 1: return combinePredicate2(predicate, predicates[0])
	default:
		for _, p := range predicates {
			predicate = combinePredicate2(predicate, p)
		}
		return predicate
	}
}

func MapSlice[T, U any](s []T, f func(T) U) []U {
	r := make([]U, len(s))
	for i, v := range s {
		r[i] = f(v)
	}
	return r
}

func FilterSlice[T any](s []T, f func(T) bool) []T {
	var r []T
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func ReduceSlice[T, U any](s []T, init U, f func(U, T) U) U {
	r := init
	for _, v := range s {
		r = f(r, v)
	}
	return r
}

func forEach(iter any, f func(i int, val any) bool) {
	v := reflect.ValueOf(iter)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i).Interface()
			if f(i, val) {
				return
			}
		}
	default:
		panic(fmt.Errorf("forEach: expected iter or array, found %q", v.Kind().String()))
	}
}
