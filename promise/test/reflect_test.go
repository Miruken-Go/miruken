package test

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/require"
	"reflect"
	"runtime"
	"testing"
)

func TestPromise_UnderlyingType(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	require.Equal(t, reflect.TypeOf(""), p1.UnderlyingType())

	p2 := promise.New(func(resolve func(int), reject func(error)) {
		resolve(22)
	})
	require.Equal(t, reflect.TypeOf(1), p2.UnderlyingType())
}

func TestPromise_UntypedPromise(t *testing.T) {
	p := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	pu := p.UntypedPromise()
	require.IsType(t, &promise.Promise[any]{}, pu)
	result, _ := pu.Await()
	require.Equal(t, "Hello", result)
}

func TestInspect(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	ut, ok := promise.Inspect(reflect.TypeOf(p1))
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(""), ut)

	p2 := promise.New(func(resolve func(int), reject func(error)) {
		resolve(22)
	})
	ut, ok = promise.Inspect(reflect.TypeOf(p2))
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(1), ut)

	ut, ok = promise.Inspect(reflect.TypeOf(1))
	require.False(t, ok)
}

func TestCoerce(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve("Hello")
	})
	pc := promise.Coerce[string](p)
	result, _ := pc.Await()
	require.Equal(t, "Hello", result)
}

func TestCoerce_Fail(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve(22)
	})
	pc := promise.Coerce[string](p)
	_, err := pc.Await()
	require.NotNil(t, err)
	var ta *runtime.TypeAssertionError
	require.ErrorAs(t, err, &ta)
	require.Equal(t, "interface conversion: interface {} is int, not string", ta.Error())
}

func TestUnwrap_Resolve(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1, func(data string) *promise.Promise[string] {
		return promise.Resolve(fmt.Sprintf("%s World", data))
	}))
	result, err := p2.Await()
	require.Nil(t, err)
	require.Equal(t, "Hello World", result)
}

func TestUnwrap_Reject(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1, func(data string) *promise.Promise[string] {
		return promise.Reject[string](fmt.Errorf("%s Error", data))
	}))
	result, err := p2.Await()
	require.Equal(t, "", result)
	require.Equal(t, "Hello Error", err.Error())
}