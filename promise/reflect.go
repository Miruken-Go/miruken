package promise

import (
	"reflect"
	"time"
)

type (
	Reflect interface {
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

func (p *Promise[T]) UnderlyingType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (p *Promise[T]) Then(
	res func(data any) any,
) *Promise[any] {
	if res == nil {
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
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
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
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
	p.value = result.(T)
}

func (p *Promise[T]) coerce(
	promise Reflect,
) {
	if p.ch == nil {
		p.ch = make(chan struct{})
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
	return Then(promise.Then(func(data any) any {
		return data
	}), func(data any) (t T) {
		if data != nil {
			t = data.(T)
		}
		return t
	})
}

func CoerceType(
	typ     reflect.Type,
	promise Reflect,
) Reflect {
	if typ.Kind() != reflect.Ptr ||  !typ.Implements(reflectType) {
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
	return New(func(resolve func(T), reject func(error)) {
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

func Delay(delay time.Duration) *Promise[any] {
	return New(func(resolve func(any), _ func(error)) {
		time.Sleep(delay)
		resolve(nil)
	})
}

var reflectType = reflect.TypeOf((*Reflect)(nil)).Elem()