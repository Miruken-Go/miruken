package promise

import "context"

// Deferred represents a computation that fulfills a Promise.
type Deferred[T any] struct {
	promise *Promise[T]
}

func (d Deferred[T]) Promise() *Promise[T]{
	return d.promise
}

func (d Deferred[T]) Resolve(resolution T) {
	d.promise.resolve(resolution)
}

func (d Deferred[T]) Reject(err error) {
	d.promise.reject(err)
}

// Defer creates a Deferred computation.
func Defer[T any]() Deferred[T] {
	p := &Promise[T]{
		ch: make(chan struct{}),
	}
	return Deferred[T]{ p }
}

// DeferWithContext creates a Deferred computation in a context.
func DeferWithContext[T any](ctx context.Context) Deferred[T] {
	d := Defer[T]()
	d.promise.ctx = ctx
	return d
}