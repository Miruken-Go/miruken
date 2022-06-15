package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type OrderStatus uint8

const (
	OrderCreated OrderStatus = 1 << iota
	OrderCancelled
)

type (
	Order struct {
		id     int
		status OrderStatus
	}

	CancelOrder struct {
		orderId int
	}

	CancelOrderFilter struct {}

	OrderHandler struct {}
)

func (c CancelOrderFilter) Order() int {
	return miruken.FilterStage
}

func (c CancelOrderFilter) AppliesTo(
	callback miruken.Callback,
) bool {
	if h, ok := callback.(*miruken.Handles); ok {
		_, ok := h.Source().(*CancelOrder)
		return ok
	}
	return false
}

func (c CancelOrderFilter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
	cancel := ctx.Callback().(*miruken.Handles).Source().(*CancelOrder)
	order  := &Order{cancel.orderId, OrderCreated}
	if err := api.StashPut(ctx.Composer(), order); err != nil {
		return next.Fail(err)
	}
	return next.Pipe()
}

func (o *OrderHandler) Cancel(
	cancel *CancelOrder,
	order  *Order,
) (*Order, error) {
	order.status = OrderCancelled
	return order, nil
}

type StashTestSuite struct {
	suite.Suite
	handler miruken.Handler
}

func (suite *StashTestSuite) SetupTest() {
	suite.handler, _ = miruken.Setup(
		//TestFeature,
		api.Feature(),
	)
}

func (suite *StashTestSuite) TestStash() {
	suite.Run("Put", func() {
		order := &Order{1, OrderCreated}
		err := api.StashPut(suite.handler, order)
		suite.Nil(err)
	})

	suite.Run("Unmanaged", func() {
		stash, _, err := miruken.Create[*api.Stash](suite.handler)
		suite.Nil(err)
		suite.Nil(stash)
	})
}

func TestStashTestSuite(t *testing.T) {
	suite.Run(t, new(StashTestSuite))
}

