package test

import (
	"context"
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

func TestLift(t *testing.T) {
	var p *promise.Promise[string]
	p = promise.Lift(reflect.TypeOf(p), "Hello").(*promise.Promise[string])
	require.NotNil(t, p)
}

func TestCoerce(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve("Hello")
	})
	pc := promise.Coerce[string](p, context.Background())
	result, _ := pc.Await(context.Background())
	require.Equal(t, "Hello", result)
}

func TestCoerce_Fail(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve(22)
	})
	pc := promise.Coerce[string](p, context.Background())
	_, err := pc.Await(context.Background())
	require.NotNil(t, err)
	var ta *runtime.TypeAssertionError
	require.ErrorAs(t, err, &ta)
	require.Equal(t, "interface conversion: interface {} is int, not string", ta.Error())
}

func TestCoerceType(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve("Hello")
	})
	var ps *promise.Promise[string]
	pc := promise.CoerceType(reflect.TypeOf(ps), p, context.Background()).(*promise.Promise[string])
	result, _ := pc.Await(context.Background())
	require.Equal(t, "Hello", result)
}

func TestCoerceType_Fail(t *testing.T) {
	p := promise.New(func(resolve func(any), reject func(error)) {
		resolve(22)
	})
	var ps *promise.Promise[string]
	pc := promise.CoerceType(reflect.TypeOf(ps), p, context.Background()).(*promise.Promise[string])
	_, err := pc.Await(context.Background())
	var ta *runtime.TypeAssertionError
	require.ErrorAs(t, err, &ta)
	require.Equal(t, "interface conversion: interface {} is int, not string", ta.Error())
}

func TestUnwrap_Resolve(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1, context.Background(),
		func(data string) *promise.Promise[string] {
			return promise.Resolve(fmt.Sprintf("%s World", data))
		}), context.Background())
	result, err := p2.Await(context.Background())
	require.Nil(t, err)
	require.Equal(t, "Hello World", result)
}

func TestUnwrap_Reject(t *testing.T) {
	p1 := promise.New(func(resolve func(string), reject func(error)) {
		resolve("Hello")
	})
	p2 := promise.Unwrap(promise.Then(p1, context.Background(),
			func(data string) *promise.Promise[string] {
			return promise.Reject[string](fmt.Errorf("%s Error", data))
		}), context.Background())
	result, err := p2.Await(context.Background())
	require.Equal(t, "", result)
	require.Equal(t, "Hello Error", err.Error())
}