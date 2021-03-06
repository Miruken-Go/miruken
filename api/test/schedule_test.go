package test

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"testing"
)

type (
	StockQuote struct {
		Symbol string
		Value  float64
	}

	GetStockQuote struct {
		Symbol string
	}

	SellStock struct {
		Symbol       string
		NumberShares int
	}

	StockQuoteHandler struct {}
)

func (s *StockQuoteHandler) Quote(
	_*miruken.Handles, quote GetStockQuote,
) *promise.Promise[StockQuote] {
	if symbol := quote.Symbol; symbol == "EX" {
		return promise.Reject[StockQuote](
			errors.New("stock exchange is down"))
	} else {
		return promise.Resolve(StockQuote{
			symbol,
			(rand.Float64()*10)+1,
		})
	}
}

func (s *StockQuoteHandler) Sell(
	_*miruken.Handles, sell SellStock,
) *promise.Promise[promise.Void] {
	if symbol := sell.Symbol; symbol == "EX" {
		return promise.Reject[promise.Void](
			errors.New("stock exchange is down"))
	}
	return promise.Resolve(promise.Void{})
}

type ScheduleTestSuite struct {
	suite.Suite
}

func (suite *ScheduleTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		api.Feature(),
	)
	return handler
}

func (suite *ScheduleTestSuite) TestSchedule() {
	suite.Run("Sequential", func() {
		suite.Run("Success", func() {
			handler := suite.Setup()
			sequential := &api.Sequential{
				Requests: []any{
					GetStockQuote{"APPL"},
					GetStockQuote{"MSFT"},
					GetStockQuote{"GOOGL"},
				},
			}
			r, pr, err := api.Send[*api.ScheduledResult](handler, sequential)
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Len(r.Responses, 3)
			for i, response := range r.Responses {
				either.Match(response,
					func(error) { panic("unexpected") },
					func(quote any) {
						suite.Equal(
							sequential.Requests[i].(GetStockQuote).Symbol,
							quote.(StockQuote).Symbol)
					})
			}
		})

		suite.Run("First Failure", func() {
			handler := suite.Setup()
			sequential := &api.Sequential{
				Requests: []any{
					GetStockQuote{"APPL"},
					GetStockQuote{"EX"},
					GetStockQuote{"EX"},
				},
			}
			r, pr, err := api.Send[*api.ScheduledResult](handler, sequential)
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Len(r.Responses, 2)
			symbols := make([]string, 2)
			for i, response := range r.Responses {
				symbols[i] = either.Fold(response,
					func(err error) string { return err.Error() },
					func(quote any) string { return quote.(StockQuote).Symbol })
			}
			suite.Equal(
				[]string { "APPL", "stock exchange is down"},
				symbols)
		})
	})

	suite.Run("Concurrent", func() {
		suite.Run("Success", func() {
			handler := suite.Setup()
			sequential := &api.Concurrent{
				Requests: []any{
					GetStockQuote{"APPL"},
					GetStockQuote{"MSFT"},
					GetStockQuote{"GOOGL"},
				},
			}
			r, pr, err := api.Send[*api.ScheduledResult](handler, sequential)
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Len(r.Responses, 3)
			symbols := make([]string, 3)
			for i, response := range r.Responses {
				symbols[i] = either.Fold(response,
					func(err error) string { return err.Error() },
					func(quote any) string { return quote.(StockQuote).Symbol })
			}
			suite.Equal(
				[]string { "APPL", "MSFT", "GOOGL"},
				symbols)
		})

		suite.Run("Single Failure", func() {
			handler := suite.Setup()
			sequential := &api.Concurrent{
				Requests: []any{
					GetStockQuote{"APPL"},
					GetStockQuote{"EX"},
					GetStockQuote{"GOOGL"},
				},
			}
			r, pr, err := api.Send[*api.ScheduledResult](handler, sequential)
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Len(r.Responses, 3)
			symbols := make([]string, 3)
			for i, response := range r.Responses {
				symbols[i] = either.Fold(response,
					func(err error) string { return err.Error() },
					func(quote any) string { return quote.(StockQuote).Symbol })
			}
			suite.Equal(
				[]string { "APPL", "stock exchange is down", "GOOGL"},
				symbols)
		})

		suite.Run("Multiple Failures", func() {
			handler := suite.Setup()
			sequential := &api.Concurrent{
				Requests: []any{
					GetStockQuote{"APPL"},
					GetStockQuote{"EX"},
					GetStockQuote{"EX"},
				},
			}
			r, pr, err := api.Send[*api.ScheduledResult](handler, sequential)
			suite.Nil(err)
			suite.NotNil(pr)
			r, err = pr.Await()
			suite.Nil(err)
			suite.Len(r.Responses, 3)
			symbols := make([]string, 3)
			for i, response := range r.Responses {
				symbols[i] = either.Fold(response,
					func(err error) string { return err.Error() },
					func(quote any) string { return quote.(StockQuote).Symbol })
			}
			suite.Equal(
				[]string { "APPL", "stock exchange is down", "stock exchange is down"},
				symbols)
		})
	})
}

func TestScheduleTestSuite(t *testing.T) {
	suite.Run(t, new(ScheduleTestSuite))
}
