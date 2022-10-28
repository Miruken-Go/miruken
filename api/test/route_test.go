package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type (
	TrashHandler struct {}
)

func (t *TrashHandler) Trash(
	_*miruken.Handles, routed api.Routed,
) (*promise.Promise[any], miruken.HandleResult) {
	if strings.EqualFold(routed.Route, "trash") {
		return promise.Resolve[any](nil), miruken.Handled
	}
	return nil, miruken.NotHandled
}

type RouteTestSuite struct {
	suite.Suite
}

func (suite *RouteTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		api.Feature(),
	)
	return handler
}

func (suite *RouteTestSuite) TestRoute() {
	suite.Run("Route", func() {
		suite.Run("Requests", func() {
			handler := suite.Setup()
			r, pr, err := api.Send[StockQuote](handler,
				api.RouteTo(GetStockQuote{"GOOGL"}, "trash"))
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Zero(r.Symbol)
			suite.Zero(r.Value)
		})

		suite.Run("No Response", func() {
			handler := suite.Setup()
			pv, err := api.Post(handler,
				api.RouteTo(SellStock{"EX", 10}, "trash"))
			suite.Nil(err)
			suite.NotNil(pv)
			_, err = pv.Await()
			suite.Nil(err)
		})

		suite.Run("Pass Through", func() {
			handler := suite.Setup()
			r, pr, err := api.Send[StockQuote](handler,
				api.RouteTo(GetStockQuote{"GOOGL"}, "pass-through"))
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Equal("GOOGL", r.Symbol)
			suite.True(r.Value > 0)
		})

		suite.Run("Unrecognized", func() {
			handler := suite.Setup()
			r, pr, err := api.Send[StockQuote](handler,
				api.RouteTo(GetStockQuote{"GOOGL"}, "NoWhere"))
			suite.Error(miruken.NotHandledError{}, err)
			suite.Nil(pr)
			suite.Zero(r.Symbol)
			suite.Zero(r.Value)
		})
	})
}

func TestRouteTestSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}
