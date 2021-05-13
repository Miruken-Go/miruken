package miruken

import (
	"errors"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandled_Or(t *testing.T) {
	t.Parallel()

	result := Handled

	t.Run("Handled should be Handled", func (t *testing.T) {
		assert.Equal(t, Handled, result.Or(Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(HandledAndStop))
	})

	t.Run("NotHandled should be Handled", func (t *testing.T) {
		assert.Equal(t, Handled, result.Or(NotHandled))
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(NotHandledAndStop))
	})
}

func TestHandledError_Or(t *testing.T) {
	t.Parallel()

	result := Handled.WithError(errors.New("bad"))

	t.Run("Handled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(Handled).WithoutError())
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(HandledAndStop).WithoutError())
	})

	t.Run("NotHandled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(NotHandled).WithoutError())
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(NotHandledAndStop).WithoutError())
	})
}

func TestHandleResultErrors(t *testing.T) {
	t.Parallel()

	t.Run("combines multiple errors", func (t *testing.T) {
		result := Handled.WithError(errors.New("bad")).
			Or(NotHandled.WithError(errors.New("argument")))

		assert.True(t, result.IsError())

		err := result.Error().(*multierror.Error)
		assert.NotNil(t, err)
		assert.Len(t, err.Errors, 2)
		assert.Equal(t, 2, len(err.Errors))
	})
}

func TestHandledAndStop_Or(t *testing.T) {
	t.Parallel()

	result := HandledAndStop

	t.Run("Handled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(HandledAndStop))
	})

	t.Run("NotHandled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(NotHandled))
	})

	t.Run("NotHandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(NotHandledAndStop))
	})
}

func TestNotHandled_Or(t *testing.T) {
	t.Parallel()

	result := NotHandled

	t.Run("Handled should be Handled", func (t *testing.T) {
		assert.Equal(t, Handled, result.Or(Handled).WithoutError())
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(HandledAndStop).WithoutError())
	})

	t.Run("NotHandled should be NotHandled", func (t *testing.T) {
		assert.Equal(t, NotHandled, result.Or(NotHandled).WithoutError())
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.Or(NotHandledAndStop).WithoutError())
	})
}

func TestNotHandledAndStop_Or(t *testing.T) {
	t.Parallel()

	result := NotHandledAndStop

	t.Run("Handled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.Or(HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.Or(NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.Or(NotHandledAndStop))
	})
}

func TestHandled_And(t *testing.T) {
	t.Parallel()

	result := Handled

	t.Run("Handled should be Handled", func (t *testing.T) {
		assert.Equal(t, Handled, result.And(Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.And(HandledAndStop))
	})

	t.Run("NotHandled should be NotHandled", func (t *testing.T) {
		assert.Equal(t, NotHandled, result.And(NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandledAndStop))
	})
}

func TestHandledAndStop_And(t *testing.T) {
	t.Parallel()

	result := HandledAndStop

	t.Run("Handled should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.And(Handled))
	})

	t.Run("HandledAndStop should be HandledAndStop", func (t *testing.T) {
		assert.Equal(t, HandledAndStop, result.And(HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandledAndStop))
	})
}

func TestNotHandled_And(t *testing.T) {
	t.Parallel()

	result := NotHandled

	t.Run("Handled should be NotHandled", func (t *testing.T) {
		assert.Equal(t, NotHandled, result.And(Handled))
	})

	t.Run("HandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(HandledAndStop))
	})

	t.Run("NotHandled should be NotHandled", func (t *testing.T) {
		assert.Equal(t, NotHandled, result.And(NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandledAndStop))
	})
}

func TestNotHandledAndStop_And(t *testing.T) {
	t.Parallel()

	result := NotHandledAndStop

	t.Run("Handled should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(Handled))
	})

	t.Run("HandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(HandledAndStop))
	})

	t.Run("NotHandled should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandled))
	})

	t.Run("NotHandledAndStop should be NotHandledAndStop", func (t *testing.T) {
		assert.Equal(t, NotHandledAndStop, result.And(NotHandledAndStop))
	})
}