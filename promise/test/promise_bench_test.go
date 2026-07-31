package test

import (
	"context"
	"testing"

	"github.com/miruken-go/miruken/promise"
)

func BenchmarkThen_AlreadyResolved(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := promise.Resolve(i)
		r := promise.Then(p, func(v int) int { return v + 1 })
		_, _ = r.Await()
	}
}

func BenchmarkThen_Async(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := promise.New(context.Background(), func(resolve func(int), _ func(error), _ func(func())) {
			resolve(i)
		})
		r := promise.Then(p, func(v int) int { return v + 1 })
		_, _ = r.Await()
	}
}

func BenchmarkCatch_AlreadyResolved(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := promise.Reject[int](errBench)
		r := promise.Catch(p, func(err error) error { return err })
		_, _ = r.Await()
	}
}

func BenchmarkCatch_Async(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := promise.New(context.Background(), func(_ func(int), reject func(error), _ func(func())) {
			reject(errBench)
		})
		r := promise.Catch(p, func(err error) error { return err })
		_, _ = r.Await()
	}
}

var errBench = context.DeadlineExceeded
