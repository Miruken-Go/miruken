package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/es"
	"github.com/miruken-go/miruken/es/test/todo"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type HandlesTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *HandlesTestSuite) SetupTest() {
	suite.specs = []any{
		&todo.List{},
	}
}

func (suite *HandlesTestSuite) Setup(specs ...any) (*context.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New(es.Feature()).
		Specs(specs...).
		Context()
}

func (suite *HandlesTestSuite) TestHandles() {
	suite.Run("Handles", func() {
		suite.Run("Default", func() {
			ctx, _ := suite.Setup()
			_, err := miruken.Command(ctx, todo.AddTask{Task: "shopping"})
			suite.Nil(err)
		})

		suite.Run("Named", func() {
			ctx, _ := suite.Setup()
			_, err := miruken.Command(ctx, todo.CompleteTasks{Tasks: []string{"shopping"}})
			suite.Nil(err)
		})
	})
}

func TestHandlesTestSuite(t *testing.T) {
	suite.Run(t, new(HandlesTestSuite))
}
