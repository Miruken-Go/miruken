package promise

import (
	"context"
	"reflect"
	"time"
)

type (
	Reflect interface {
		Context() context.Context
		UnderlyingType() reflect.Type
		Then(resolve func(data any) any) *Promise[any]
		Catch(reject func(err error) error) *Promise[any]
		AwaitAny() (any, error)
	}

	internal interface {
		Reflect
		lift(result any)
		coerce(promise Reflect)
	}
)

func (p *Promise[T]) Context() context.Context {
	return p.ctx
}

func (p *Promise[T]) UnderlyingType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (p *Promise[T]) Then(
	res func(data any) any,
) *Promise[any] {
	if res == nil {
		panic("res cannot be nil")
	}
	return New(p.ctx, func(resolve func(any), reject func(error), onCancel func(func())) {
		result, err := p.Await()
		if err != nil {
			reject(err)
			return
		}
		resolve(res(result))
	})
}

func (p *Promise[T]) Catch(
	rej func(err error) error,
) *Promise[any] {
	if rej == nil {
		panic("rej cannot be nil")
	}
	return New(p.ctx, func(resolve func(any), reject func(error), onCancel func(func())) {
		result, err := p.Await()
		if err != nil {
			reject(rej(err))
			return
		}
		resolve(result)
	})
}

func (p *Promise[T]) AwaitAny() (any, error) {
	return p.Await()
}

func (p *Promise[T]) lift(result any) {
	p.resolve(result.(T))
}

func (p *Promise[T]) coerce(
	promise Reflect,
) {
	if p.ch == nil {
		p.ch = make(chan struct{})
	}
	if p.ctx == nil {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}
	promise.Then(func(result any) any {
		var t T
		if result != nil {
			t = result.(T)
		}
		p.resolve(t)
		return nil
	}).Catch(func(err error) error {
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
	if typ.Kind() != reflect.Ptr || !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	promise := reflect.New(typ.Elem()).Interface().(internal)
	promise.lift(result)
	return promise
}

func Coerce[T any](
	promise Reflect,
) *Promise[T] {
	return New(promise.Context(), func(resolve func(T), reject func(error), onCancel func(func())) {
		data, err := promise.AwaitAny()
		if err != nil {
			reject(err)
		} else {
			if data != nil {
				resolve(data.(T))
			} else {
				var t T
				resolve(t)
			}
		}
	})
}

func CoerceType(
	typ reflect.Type,
	promise Reflect,
) Reflect {
	if typ.Kind() != reflect.Ptr || !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	p := reflect.New(typ.Elem()).Interface().(internal)
	p.coerce(promise)
	return p
}

func Unwrap[T any](
	promise *Promise[*Promise[T]],
) *Promise[T] {
	if promise == nil {
		panic("promise cannot be nil")
	}
	return New(nil, func(resolve func(T), reject func(error), onCancel func(func())) {
		if pt, err := promise.Await(); err != nil {
			reject(err)
		} else {
			if data, err := pt.Await(); err != nil {
				reject(err)
			} else {
				resolve(data)
			}
		}
	})
}

func Empty() *Promise[struct{}] {
	return Resolve(struct{}{})
}

func RejectEmpty(err error) *Promise[struct{}] {
	return Reject[struct{}](err)
}

func Return[A, B any](p *Promise[A], val B) *Promise[B] {
	return Then(p, func(A) B {
		return val
	})
}

func IndirectReturn[A, B any](p *Promise[A], val *B) *Promise[B] {
	return Then(p, func(A) B {
		return *val
	})
}

func Slice[A any](p *Promise[A]) *Promise[[]A] {
	return Then(p, func(a A) []A {
		return []A{a}
	})
}

func Erase[A any](p *Promise[A]) *Promise[struct{}] {
	return Return(p, struct{}{})
}

func Delay[T any](
	ctx   context.Context,
	delay time.Duration,
) *Promise[T] {
	return New(ctx, func(resolve func(T), _ func(error), onCancel func(func())) {
		time.Sleep(delay)
		var t T
		resolve(t)
	})
}

var reflectType = reflect.TypeOf((*Reflect)(nil)).Elem()
