package promise

import (
	"context"
	"reflect"
	"time"
)

type (
	Reflect interface {
		UnderlyingType() reflect.Type
		Then(ctx context.Context, resolve func(data any) any) *Promise[any]
		Catch(ctx context.Context, reject func(err error) error) *Promise[any]
		AwaitAny(ctx context.Context) (any, error)
	}

	internal interface {
		Reflect
		lift(result any)
		coerce(promise *Promise[any], ctx context.Context)
	}
)

func (p *Promise[T]) UnderlyingType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (p *Promise[T]) Then(
	ctx context.Context,
	res func(data any) any,
) *Promise[any] {
	if res == nil {
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
		result, err := p.Await(ctx)
		if err != nil {
			reject(err)
			return
		}
		resolve(res(result))
	})
}

func (p *Promise[T]) Catch(
	ctx context.Context,
	rej func(err error) error,
) *Promise[any] {
	if rej == nil {
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
		result, err := p.Await(ctx)
		if err != nil {
			reject(rej(err))
			return
		}
		resolve(result)
	})
}

func (p *Promise[T]) AwaitAny(
	ctx context.Context,
) (any, error) {
	return p.Await(ctx)
}

func (p *Promise[T]) lift(result any) {
	if p.ch == nil {
		p.ch = make(chan struct{})
	}
	p.resolve(result.(T))
}

func (p *Promise[T]) coerce(
	promise *Promise[any],
	ctx     context.Context,
) {
	if p.ch == nil {
		p.ch = make(chan struct{})
	}
	promise.Then(ctx, func(result any) any {
		p.resolve(result.(T))
		return nil
	}).Catch(ctx, func(err error) error {
		p.reject(err)
		return nil
	})
}

func Inspect(typ reflect.Type) (reflect.Type, bool) {
	if typ != nil && typ.Implements(reflectType) {
		promise := reflect.Zero(typ).Interface().(Reflect)
		return promise.UnderlyingType(), true
	}
	return nil, false
}

func Lift(typ reflect.Type, result any) Reflect {
	if typ.Kind() != reflect.Ptr {
		panic("typ must be a promise")
	}
	if !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	promise := reflect.New(typ.Elem()).Interface().(internal)
	promise.lift(result)
	return promise
}

func Coerce[T any](
	promise *Promise[any],
	ctx     context.Context,
) *Promise[T] {
	return Then(promise, ctx, func(data any) T {
		return data.(T)
	})
}

func CoerceType(
	typ     reflect.Type,
	promise *Promise[any],
	ctx     context.Context,
) Reflect {
	if typ.Kind() != reflect.Ptr ||  !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	p := reflect.New(typ.Elem()).Interface().(internal)
	p.coerce(promise, ctx)
	return p
}

func Unwrap[T any](
	promise *Promise[*Promise[T]],
	ctx     context.Context,
) *Promise[T] {
	if promise == nil {
		panic("promise cannot be nil")
	}
	return New(func(resolve func(T), reject func(error)) {
		if pt, err := promise.Await(ctx); err != nil {
			reject(err)
		} else {
			if data, err := pt.Await(ctx); err != nil {
				reject(err)
			} else {
				resolve(data)
			}
		}
	})
}

func Delay(delay time.Duration) *Promise[any] {
	return New(func(resolve func(any), _ func(error)) {
		time.Sleep(delay)
		resolve(nil)
	})
}

var reflectType = reflect.TypeOf((*Reflect)(nil)).Elem()