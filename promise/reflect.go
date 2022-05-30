package promise

import (
	"reflect"
)

type Reflect interface {
	UnderlyingType() reflect.Type
	UntypedPromise() *Promise[any]
	Then(resolve func(data any) any) *Promise[any]
	Catch(rejection func(err error) error) *Promise[any]
}

func (p *Promise[T]) UnderlyingType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (p *Promise[T]) UntypedPromise() *Promise[any] {
	if pa, ok := (any(p)).(*Promise[any]); ok {
		return pa
	}
	return Then(p, func(data T) any {
		return data
	})
}

func (p *Promise[T]) Then(resolution func(data any) any) *Promise[any] {
	if resolution == nil {
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
		result, err := p.Await()
		if err != nil {
			reject(err)
			return
		}
		resolve(resolution(result))
	})
}

func (p *Promise[T]) Catch(rejection func(err error) error) *Promise[any] {
	if rejection == nil {
		panic("resolve cannot be nil")
	}
	return New(func(resolve func(any), reject func(error)) {
		result, err := p.Await()
		if err != nil {
			reject(rejection(err))
			return
		}
		resolve(result)
	})
}

func Inspect(typ reflect.Type) (reflect.Type, bool) {
	if typ != nil && typ.Implements(_reflectType) {
		promise := reflect.Zero(typ).Interface().(Reflect)
		return promise.UnderlyingType(), true
	}
	return nil, false
}

func Coerce[T any](promise *Promise[any]) *Promise[T] {
	return Then(promise, func(data any) T {
		return data.(T)
	})
}

func Unwrap[T any](promise *Promise[*Promise[T]]) *Promise[T] {
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

var _reflectType = reflect.TypeOf((*Reflect)(nil)).Elem()