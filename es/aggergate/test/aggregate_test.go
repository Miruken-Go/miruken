package test

import (
	"testing"

	"github.com/miruken-go/miruken/es/aggergate/test/todo"
	"github.com/miruken-go/miruken/provides"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type AggregateTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *AggregateTestSuite) SetupTest() {
	suite.specs = []any{
		&todo.List{},
	}
}

func (suite *AggregateTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *AggregateTestSuite) TestRoot() {
	suite.Run("Resolve", func() {
		suite.Run("Contextual", func() {
			ctx, _ := suite.Setup()
			list1, _, ok, err := provides.Type[*todo.List](ctx)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(list1)
			list2, _, ok, err := provides.Type[*todo.List](ctx)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(list1, list2)
		})
	})
}

func TestAggregateTestSuite(t *testing.T) {
	suite.Run(t, new(AggregateTestSuite))
}
