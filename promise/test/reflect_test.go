package test

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/require"
)

func TestPromise_UnderlyingType(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	require.Equal(t, reflect.TypeOf(""), p1.UnderlyingType())

	p2 := promise.New(nil, func(resolve func(int), reject func(error), onCancel func(func())) {
		resolve(22)
	})
	require.Equal(t, reflect.TypeOf(1), p2.UnderlyingType())
}

func TestInspect(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	ut, ok := promise.Inspect(reflect.TypeOf(p1))
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(""), ut)

	p2 := promise.New(nil, func(resolve func(int), reject func(error), onCancel func(func())) {
		resolve(22)
	})
	ut, ok = promise.Inspect(reflect.TypeOf(p2))
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf(1), ut)

	ut, ok = promise.Inspect(reflect.TypeOf(1))
	require.False(t, ok)
}

func TestLift(t *testing.T) {
	var p *promise.Promise[string]
	p = promise.Lift(reflect.TypeOf(p), "Hello").(*promise.Promise[string])
	require.NotNil(t, p)
}

func TestCoerce(t *testing.T) {
	p := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	pc := promise.Coerce[string](p)
	result, _ := pc.Await()
	require.Equal(t, "Hello", result)
}

func TestCoerce_Fail(t *testing.T) {
	p := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve(22)
	})
	pc := promise.Coerce[string](p)
	_, err := pc.Await()
	require.NotNil(t, err)
	var ta *runtime.TypeAssertionError
	require.ErrorAs(t, err, &ta)
	require.Equal(t, "interface conversion: interface {} is int, not string", ta.Error())
}

func TestCoerceType(t *testing.T) {
	p := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	var ps *promise.Promise[string]
	pc := promise.CoerceType(reflect.TypeOf(ps), p).(*promise.Promise[string])
	result, _ := pc.Await()
	require.Equal(t, "Hello", result)
}

func TestCoerceType_Fail(t *testing.T) {
	p := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve(22)
	})
	var ps *promise.Promise[string]
	pc := promise.CoerceType(reflect.TypeOf(ps), p).(*promise.Promise[string])
	_, err := pc.Await()
	var ta *runtime.TypeAssertionError
	require.ErrorAs(t, err, &ta)
	require.Equal(t, "interface conversion: interface {} is int, not string", ta.Error())
}

func TestUnwrap_Resolve(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1,
		func(data string) *promise.Promise[string] {
			return promise.Resolve(fmt.Sprintf("%s World", data))
		}))
	result, err := p2.Await()
	require.Nil(t, err)
	require.Equal(t, "Hello World", result)
}

func TestUnwrap_Reject(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1,
		func(data string) *promise.Promise[string] {
			return promise.Reject[string](fmt.Errorf("%s Error", data))
		}))
	result, err := p2.Await()
	require.Equal(t, "", result)
	require.Equal(t, "Hello Error", err.Error())
}
