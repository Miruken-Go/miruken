package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/es"
	"github.com/miruken-go/miruken/es/command"
	"github.com/miruken-go/miruken/es/test/todo"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type CommandTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *CommandTestSuite) SetupTest() {
	suite.specs = []any{
		&todo.List{},
	}
}

func (suite *CommandTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New(es.Feature()).
		Specs(specs...).
		Context()
}

func (suite *CommandTestSuite) TestCommand() {
	suite.Run("Handle", func() {
		suite.Run("Default", func() {
			ctx, _ := suite.Setup()
			_, err := miruken.Command(ctx, todo.AddTask{Task: "shopping"})
			suite.Nil(err)
		})

		/*
		suite.Run("Constrained", func() {
			ctx, _ := suite.Setup()
			_, err := miruken.Command(ctx, todo.RemoveTask{Task: "shopping"}, command.Named("todo.removeTask"))
			suite.Nil(err)
		})
		*/

		suite.Run("Overridden", func() {
			ctx, _ := suite.Setup()
			_, err := miruken.Command(ctx, todo.CompleteTasks{Tasks: []string{"shopping"}}, command.Named("completeTasks"))
			suite.Nil(err)
		})
	})
}

func TestCommandTestSuite(t *testing.T) {
	suite.Run(t, new(CommandTestSuite))
}
