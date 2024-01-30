package test

import (
	"errors"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/assert"
)

func TestHandled_Or(t *testing.T) {
	t.Parallel()

	result := miruken.Handled

	t.Run("Handled should be Handled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.Handled, result.Or(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be Handled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.Handled, result.Or(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.NotHandledAndStop))
	})
}

func TestHandledError_Or(t *testing.T) {
	t.Parallel()

	result := miruken.Handled.WithError(errors.New("bad"))

	t.Run("Handled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.Handled).WithoutError())
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.HandledAndStop).WithoutError())
	})

	t.Run("NotHandled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.NotHandled).WithoutError())
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.NotHandledAndStop).WithoutError())
	})
}

func TestHandleResultErrors(t *testing.T) {
	t.Parallel()

	t.Run("combines multiple errors", func(t *testing.T) {
		err1 := errors.New("bad")
		err2 := errors.New("argument")
		result := miruken.Handled.WithError(err1).
			Or(miruken.NotHandled.WithError(err2))

		assert.True(t, result.IsError())
		assert.ErrorIs(t, result.Error(), err1)
		assert.ErrorIs(t, result.Error(), err2)
	})
}

func TestHandledAndStop_Or(t *testing.T) {
	t.Parallel()

	result := miruken.HandledAndStop

	t.Run("Handled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.NotHandledAndStop))
	})
}

func TestNotHandled_Or(t *testing.T) {
	t.Parallel()

	result := miruken.NotHandled

	t.Run("Handled should be Handled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.Handled, result.Or(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandled, result.Or(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.Or(miruken.NotHandledAndStop))
	})
}

func TestNotHandledAndStop_Or(t *testing.T) {
	t.Parallel()

	result := miruken.NotHandledAndStop

	t.Run("Handled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.Or(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.Or(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.Or(miruken.NotHandledAndStop))
	})
}

func TestHandled_And(t *testing.T) {
	t.Parallel()

	result := miruken.Handled

	t.Run("Handled should be Handled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.Handled, result.And(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.And(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandled, result.And(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandledAndStop))
	})
}

func TestHandledAndStop_And(t *testing.T) {
	t.Parallel()

	result := miruken.HandledAndStop

	t.Run("Handled should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.And(miruken.Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.HandledAndStop, result.And(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandledAndStop))
	})
}

func TestNotHandled_And(t *testing.T) {
	t.Parallel()

	result := miruken.NotHandled

	t.Run("Handled should be NotHandled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandled, result.And(miruken.Handled))
	})

	t.Run("HandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandled", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandled, result.And(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandledAndStop))
	})
}

func TestNotHandledAndStop_And(t *testing.T) {
	t.Parallel()

	result := miruken.NotHandledAndStop

	t.Run("Handled should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.Handled))
	})

	t.Run("HandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, miruken.NotHandledAndStop, result.And(miruken.NotHandledAndStop))
	})
}
