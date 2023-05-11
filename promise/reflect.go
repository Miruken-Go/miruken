package promise

import (
	"reflect"
	"sync"
	"time"
)

type Reflect interface {
	UnderlyingType() reflect.Type

	Then(resolve func(data any) any) *Promise[any]
	Catch(reject func(err error) error) *Promise[any]
	AwaitAny() (any, error)

	Lift(result any)
	Coerce(promise *Promise[any])
}

func (p *Promise[T]) UnderlyingType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (p *Promise[T]) Then(res func(data any) any) *Promise[any] {
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

func (p *Promise[T]) Catch(rej func(err error) error) *Promise[any] {
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

func (p *Promise[T]) Lift(result any) {
	p.mutex  = &sync.Mutex{}
	p.wg     = &sync.WaitGroup{}
	p.result = result.(T)
}

func (p *Promise[T]) Coerce(promise *Promise[any]) {
	p.pending = true
	p.mutex   = &sync.Mutex{}
	p.wg      = &sync.WaitGroup{}

	p.wg.Add(1)

	promise.Then(func(result any) any {
		p.resolve(result.(T))
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
	if typ.Kind() != reflect.Ptr {
		panic("typ must be a promise")
	}
	if !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	promise := reflect.New(typ.Elem()).Interface().(Reflect)
	promise.Lift(result)
	return promise
}

func Coerce[T any](promise *Promise[any]) *Promise[T] {
	return Then(promise, func(data any) T {
		return data.(T)
	})
}

func CoerceType(typ reflect.Type, promise *Promise[any]) Reflect {
	if typ.Kind() != reflect.Ptr {
		panic("typ must be a promise")
	}
	if !typ.Implements(reflectType) {
		panic("typ must be a promise")
	}
	p := reflect.New(typ.Elem()).Interface().(Reflect)
	p.Coerce(promise)
	return p
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

func Delay(delay time.Duration) *Promise[any] {
	return New(func(resolve func(any), _ func(error)) {
		time.Sleep(delay)
		resolve(nil)
	})
}

var reflectType = reflect.TypeOf((*Reflect)(nil)).Elem()