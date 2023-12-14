package promise

import (
	"context"
	"fmt"
	"sync"
)

// This code was lifted from https://github.com/chebyrash/promise and modified
// to provide runtime support since Go Generics offer limited inspection.

// Promise represents the eventual completion (or failure) of an asynchronous operation and its resulting value
type Promise[T any] struct {
	value T
	err   error
	ctx   context.Context
	ch    chan struct{}
	once  sync.Once
}

func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	return WithContext[T](nil, executor)
}

func WithContext[T any](ctx context.Context, executor func(resolve func(T), reject func(error))) *Promise[T] {
	if executor == nil {
		panic("missing executor")
	}

	p := &Promise[T]{
		ctx: ctx,
		ch:  make(chan struct{}),
	}

	go func() {
		defer p.handlePanic()
		executor(p.resolve, p.reject)
	}()

	return p
}

func Then[A, B any](p *Promise[A], resolve func(A) B) *Promise[B] {
	return WithContext(p.ctx, func(internalResolve func(B), reject func(error)) {
		result, err := p.Await()
		if err != nil {
			reject(err)
		} else {
			internalResolve(resolve(result))
		}
	})
}

func Catch[T any](p *Promise[T], reject func(err error) error) *Promise[T] {
	return WithContext(p.ctx, func(resolve func(T), internalReject func(error)) {
		result, err := p.Await()
		if err != nil {
			internalReject(reject(err))
		} else {
			resolve(result)
		}
	})
}

func (p *Promise[T]) Await() (T, error) {
	if ch := p.ch; ch != nil {
		if ctx := p.ctx; ctx != nil {
			select {
			case <-ctx.Done():
				p.reject(CanceledError{context.Cause(ctx)})
			case <-ch:
			}
		} else {
			<-ch
		}
	}
	return p.value, p.err
}

func (p *Promise[T]) resolve(value T) {
	p.once.Do(func() {
		p.value = value
		close(p.ch)
	})
}

func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.ch)
	})
}

func (p *Promise[T]) handlePanic() {
	err := recover()
	if err == nil {
		return
	}

	switch v := err.(type) {
	case error:
		p.reject(v)
	default:
		p.reject(fmt.Errorf("%+v", v))
	}
}

// Resolve creates a Promise in the resolved state.
func Resolve[T any](value T) *Promise[T] {
	return &Promise[T]{value: value}
}

// Reject creates a Promise in the rejected state.
func Reject[T any](err error) *Promise[T] {
	return &Promise[T]{err: err}
}

// All resolves when all promises have resolved, or rejects immediately upon any of the promises rejecting
func All[T any](
	promises ...*Promise[T],
) *Promise[[]T] {
	if len(promises) == 0 {
		panic("at lease one promise required")
	}

	return New(func(resolve func([]T), reject func(error)) {
		resultsChan := make(chan tuple[T, int], len(promises))
		errsChan := make(chan error, len(promises))

		for idx, p := range promises {
			idx := idx
			_ = Then(p, func(data T) T {
				resultsChan <- tuple[T, int]{_1: data, _2: idx}
				return data
			})
			_ = Catch(p, func(err error) error {
				errsChan <- err
				return err
			})
		}

		results := make([]T, len(promises))
		for idx := 0; idx < len(promises); idx++ {
			select {
			case result := <-resultsChan:
				results[result._2] = result._1
			case err := <-errsChan:
				reject(err)
				return
			}
		}
		resolve(results)
	})
}

// Race resolves or rejects as soon as any one of the promises resolves or rejects
func Race[T any](
	promises ...*Promise[T],
) *Promise[T] {
	if len(promises) == 0 {
		panic("missing promises")
	}

	return New(func(resolve func(T), reject func(error)) {
		valsChan := make(chan T, len(promises))
		errsChan := make(chan error, len(promises))

		for _, p := range promises {
			_ = Then(p, func(data T) T {
				valsChan <- data
				return data
			})
			_ = Catch(p, func(err error) error {
				errsChan <- err
				return err
			})
		}

		select {
		case val := <-valsChan:
			resolve(val)
		case err := <-errsChan:
			reject(err)
		}
	})
}

type tuple[T1, T2 any] struct {
	_1 T1
	_2 T2
}

type CanceledError struct {
	cause error
}

func (e CanceledError) Cause() error {
	return e.cause
}

func (e CanceledError) Error() string {
	if cause := e.cause; cause != nil {
		return "promise: canceled: " + cause.Error()
	}
	return "promise: canceled"
}

func (e CanceledError) Unwrap() error {
	return e.cause
}
