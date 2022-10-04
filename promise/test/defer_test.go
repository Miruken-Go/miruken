package test

import (
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPromise_Defer(t *testing.T) {
	d := promise.Defer[string]()
	require.NotNil(t, d.Promise())
}

func TestDeferred_Resolve(t *testing.T) {
	d := promise.Defer[string]()
	p := promise.Then(d.Promise(), func(data string) string {
		return data + ", world!"
	})

	d.Resolve("Hello")
	val, err := p.Await()
	require.NoError(t, err)
	require.Equal(t, val, "Hello, world!")
}

func TestDeferred_Catch(t *testing.T) {
	d := promise.Defer[string]()
	p := promise.Then(d.Promise(), func(data string) any {
		t.Fatal("should not execute Then")
		return nil
	})

	d.Reject(expectedError)
	val, err := p.Await()
	require.Error(t, err)
	require.Equal(t, expectedError, err)
	require.Nil(t, val)
}