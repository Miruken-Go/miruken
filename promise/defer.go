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
		pending: true,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}

	p.wg.Add(1)

	return Deferred[T]{ p }
}