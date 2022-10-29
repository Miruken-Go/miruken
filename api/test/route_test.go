package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	Trash struct {
		items []any
	}

	TrashHandler struct {}
)

func (t *Trash) Add(items ...any) {
	t.items = append(t.items, items...)
}

func (t *Trash) Items() []any {
	return t.items
}

func (t *Trash) Empty() {
	t.items = nil
}

func (t *TrashHandler) Trash(
	_*struct{
		miruken.Handles
		api.Routes `scheme:"trash"`
	  }, routed api.Routed,
	trash *Trash,
) *promise.Promise[any] {
	trash.Add(routed.Message)
	return promise.Resolve[any](nil)
}

type RouteTestSuite struct {
	suite.Suite
}

func (suite *RouteTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		api.Feature(),
		miruken.HandlerSpecs(&Trash{}),
	)
	return handler
}

func (suite *RouteTestSuite) TestRoute() {
	suite.Run("Route", func() {
		suite.Run("Requests", func() {
			handler := suite.Setup()
			trash, _, _ := miruken.Resolve[*Trash](handler)
			getQuote := GetStockQuote{"GOOGL"}
			r, pr, err := api.Send[StockQuote](handler, api.RouteTo(getQuote, "trash"))
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Zero(r.Symbol)
			suite.Zero(r.Value)
			items := trash.Items()
			suite.Len(items, 1)
			suite.Equal(getQuote, items[0])
		})

		suite.Run("No Response", func() {
			handler := suite.Setup()
			trash, _, _ := miruken.Resolve[*Trash](handler)
			sell := SellStock{"EX", 10}
			pv, err := api.Post(handler, api.RouteTo(sell, "trash"))
			suite.Nil(err)
			suite.NotNil(pv)
			_, err = pv.Await()
			suite.Nil(err)
			items := trash.Items()
			suite.Len(items, 1)
			suite.Equal(sell, items[0])
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

	suite.Run("Route Batch", func() {
		suite.Run("Requests", func() {
			handler := suite.Setup()
			trash, _, _ := miruken.Resolve[*Trash](handler)
			getQuote := GetStockQuote{"GOOGL"}
			pb := miruken.Batch(handler, func(batch miruken.Handler) {
				_, _, err := api.Send[StockQuote](batch, api.RouteTo(getQuote, "trash"))
				suite.Nil(err)
			})
			results, err := pb.Await()
			suite.Nil(err)
			suite.NotNil(results)
			items := trash.Items()
			suite.Len(items, 1)
			suite.Equal(api.ConcurrentBatch{Requests: []any{getQuote}}, items[0])
		})
	})
}

func TestRouteTestSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}
