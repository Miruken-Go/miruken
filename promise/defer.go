package promise

import "sync"

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
		value: nil,
		err:   nil,
		ch:    make(chan struct{}),
		once:  sync.Once{},
	}
	return Deferred[T]{ p }
}