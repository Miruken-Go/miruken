package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"sync/atomic"
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
		handles.It
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

func (suite *RouteTestSuite) Setup() *context.Context {
	ctx, _ := setup.New(
		TestFeature,
		api.Feature()).
		Specs(&Trash{}).
		Context()
	return ctx
}

func (suite *RouteTestSuite) TestRoute() {
	suite.Run("Route", func() {
		suite.Run("Requests", func() {
			handler := suite.Setup()
			trash, _, _, _ := provides.Type[*Trash](handler)
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
			trash, _, _, _ := provides.Type[*Trash](handler)
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
			getQuote := api.RouteTo(GetStockQuote{"GOOGL"}, "NoWhere")
			r, pr, err := api.Send[StockQuote](handler, getQuote)
			suite.IsType(err, &miruken.NotHandledError{})
			suite.Nil(pr)
			suite.Zero(r.Symbol)
			suite.Zero(r.Value)
		})
	})

	suite.Run("Route Batch", func() {
		suite.Run("Requests", func() {
			handler := suite.Setup()
			trash, _, _, _ := provides.Type[*Trash](handler)
			getQuote := GetStockQuote{"GOOGL"}
			called := false
			pb := miruken.BatchAsync(handler,
				func(batch miruken.Handler) *promise.Promise[any] {
					_, pq, err := api.Send[StockQuote](batch, api.RouteTo(getQuote, "trash"))
					suite.Nil(err)
					return pq.Catch(func(err error) error {
						suite.Equal(err, api.ErrMissingResponse)
						called = true
						return nil
					})
			})
			results, err := pb.Await()
			suite.Nil(err)
			suite.True(called)
			suite.Len(results, 1)
			suite.Equal([]any{api.RouteReply{Uri: "trash", Responses: []any{}}}, results[0])
			items := trash.Items()
			suite.Len(items, 1)
			suite.Equal(api.ConcurrentBatch{Requests: []any{getQuote}}, items[0])
		})

		suite.Run("Pass Through", func() {
			handler := suite.Setup()
			var counter int32
			pb := miruken.BatchAsync(handler,
				func(batch miruken.Handler) *promise.Promise[[]StockQuote] {
					_, pq1, err1 := api.Send[StockQuote](batch,
						api.RouteTo(GetStockQuote{"GOOGL"}, "pass-through"))
					suite.Nil(err1)
					p1 := promise.Then(pq1, func(quote StockQuote) StockQuote {
						suite.Equal("GOOGL", quote.Symbol)
						atomic.AddInt32(&counter, 1)
						return quote
					})
					_, pq2, err2 := api.Send[StockQuote](batch,
						api.RouteTo(GetStockQuote{"APPL"}, "pass-through"))
					suite.Nil(err2)
					p2 := promise.Then(pq2, func(quote StockQuote) StockQuote {
						suite.Equal("APPL", quote.Symbol)
						atomic.AddInt32(&counter, 1)
						return quote
					})
					return promise.All(p1, p2)
			})
			results, err := pb.Await()
			suite.Nil(err)
			suite.Equal(int32(2), counter)
			suite.Len(results, 1)
			groups := slices.OfType[any, []any](results)
			suite.Len(groups, 1)
			replies := slices.OfType[any, api.RouteReply](groups[0])
			suite.Len(replies, 1)
			suite.Equal("pass-through", replies[0].Uri)
			suite.Len(replies[0].Responses, 2)
		})
	})
}

func TestRouteTestSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}
