package test

import (
	"testing"

	"github.com/miruken-go/miruken/es/aggergate/test/todo"

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
}

func TestAggregateTestSuite(t *testing.T) {
	suite.Run(t, new(AggregateTestSuite))
}
