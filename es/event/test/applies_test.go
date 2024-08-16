package test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/es"
	"github.com/miruken-go/miruken/es/event"
	"github.com/miruken-go/miruken/es/test/todo"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type AppliesTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *AppliesTestSuite) SetupTest() {
	suite.specs = []any{
		&todo.List{},
	}
}

func (suite *AppliesTestSuite) Setup(specs ...any) (*context.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New(es.Feature()).
		Specs(specs...).
		Context()
}

func (suite *AppliesTestSuite) TestApplies() {
	suite.Run("Applies", func() {
		suite.Run("Default", func() {
			ctx, _ := suite.Setup()
			l := &todo.List{Id: uuid.UUID{}, Version: 1}
			ctx.AppendHandlers(l)
			err := event.Apply(ctx, todo.TaskAdded{Task: "shopping"})
			suite.Nil(err)
			suite.ElementsMatch([]string{"shopping"}, l.Tasks())
		})

		suite.Run("NoInference", func() {
			ctx, _ := suite.Setup()
			err := event.Apply(ctx, todo.TaskAdded{Task: "shopping"})
			suite.NotNil(err)
			var nh *miruken.NotHandledError
			suite.ErrorAs(err, &nh)
		})
	})
}

func TestAppliesTestSuite(t *testing.T) {
	suite.Run(t, new(AppliesTestSuite))
}
