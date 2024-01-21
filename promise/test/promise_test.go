package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/miruken-go/miruken/promise"

	"github.com/stretchr/testify/require"
)

var errExpected = errors.New("expected error")

func TestNew(t *testing.T) {
	p := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve(nil)
	})
	require.NotNil(t, p)
}

func TestPromise_Then(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("Hello, ")
	})
	p2 := promise.Then(p1, func(data string) string {
		return data + "world!"
	})

	val, err := p1.Await()
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "Hello, ", val)

	val, err = p2.Await()
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "Hello, world!", val)
}

func TestPromise_Catch(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})
	p2 := promise.Then(p1, func(data any) any {
		t.Fatal("should not execute Then")
		return nil
	})

	val, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, errExpected, err)
	require.Nil(t, val)

	_, _ = p2.Await()
}

func TestPromise_Panic(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		panic("random error")
	})
	p2 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		panic(errExpected)
	})

	val, err := p1.Await()
	require.Error(t, err)
	require.Equal(t, errors.New("random error"), err)
	require.Nil(t, val)

	val, err = p2.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestAll_Happy(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("one")
	})
	p2 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("two")
	})
	p3 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("three")
	})

	p := promise.All(nil, p1, p2, p3)

	val, err := p.Await()
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, []string{"one", "two", "three"}, val)
}

func TestAll_ContainsRejected(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("one")
	})
	p2 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})
	p3 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		resolve("three")
	})

	p := promise.All(nil, p1, p2, p3)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestAll_OnlyRejected(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})
	p2 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})
	p3 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})

	p := promise.All(nil, p1, p2, p3)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestRace_Happy(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		time.Sleep(time.Millisecond * 100)
		resolve("faster")
	})
	p2 := promise.New(nil, func(resolve func(string), reject func(error), onCancel func(func())) {
		time.Sleep(time.Millisecond * 500)
		resolve("slower")
	})

	p := promise.Race(nil, p1, p2)

	val, err := p.Await()
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "faster", val)
}

func TestRace_ContainsRejected(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		time.Sleep(time.Millisecond * 100)
		resolve(nil)
	})
	p2 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})

	p := promise.Race(nil, p1, p2)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestRace_OnlyRejected(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})
	p2 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		reject(errExpected)
	})

	p := promise.Race(nil, p1, p2)

	val, err := p.Await()
	require.Error(t, err)
	require.ErrorIs(t, err, errExpected)
	require.Nil(t, val)
}

func TestPromise_CancelContext(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute*10))
	p1 := promise.New(ctx, func(resolve func(any), reject func(error), onCancel func(func())) {})
	cancel()

	val, err := p1.Await()
	require.Error(t, err)
	var canceled promise.CanceledError
	require.ErrorAs(t, err, &canceled)
	require.Equal(t, context.Canceled, canceled.Cause())
	require.Nil(t, val)
}

func TestPromise_Cancel(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {})
	p1.Cancel()

	val, err := p1.Await()
	require.Error(t, err)
	var canceled promise.CanceledError
	require.ErrorAs(t, err, &canceled)
	require.Equal(t, context.Canceled, canceled.Cause())
	require.Nil(t, val)
}

func TestPromise_Foo(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute*10))
	p1 := promise.New(ctx, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve("craig")
	})
	val, err := p1.Await()
	require.Nil(t, err)
	require.Equal(t, "craig", val)

	cancel()

	val, err = p1.Await()
	require.Nil(t, err)
	require.Equal(t, "craig", val)
}

func TestPromise_Timeout(t *testing.T) {
	p1 := promise.New(nil, func(resolve func(any), reject func(error), onCancel func(func())) {
		resolve("Hello")
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	p2 := promise.New(ctx, func(resolve func(any), reject func(error), onCancel func(func())) {})
	defer cancel()

	val, err := p1.Await()
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, "Hello", val)

	val, err = p2.Await()
	require.Error(t, err)
	var canceled promise.CanceledError
	require.ErrorAs(t, err, &canceled)
	require.Equal(t, context.DeadlineExceeded, canceled.Cause())
	require.Nil(t, val)
}
