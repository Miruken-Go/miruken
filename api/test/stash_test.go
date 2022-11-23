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

func (c *CancelOrderFilter) Order() int {
	return miruken.FilterStage
}

func (c *CancelOrderFilter) Next(
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
	_*struct{
		miruken.Handles
		CancelOrderFilter
      }, cancel *CancelOrder,
	order  *Order,
) (*Order, error) {
	order.status = OrderCancelled
	return order, nil
}

type StashTestSuite struct {
	suite.Suite
}

func (suite *StashTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		api.Feature(),
	)
	return handler
}

func (suite *StashTestSuite) TestStash() {
	suite.Run("Unmanaged", func() {
		handler := suite.Setup()
		_, _, err := miruken.Create[*api.Stash](handler)
		suite.IsType(err, &miruken.NotHandledError{})
	})

	suite.Run("Put", func() {
		handler := suite.Setup()
		order := &Order{1, OrderCreated}
		err := api.StashPut(handler, order)
		suite.Nil(err)
		o, ok := api.StashGet[*Order](handler)
		suite.True(ok)
		suite.Same(order, o)
	})

	suite.Run("GetOrPut", func() {
		handler := suite.Setup()
		order := &Order{1, OrderCreated}
		o, err := api.StashGetOrPut(handler, order)
		suite.Nil(err)
		suite.Same(order, o)
		o, ok := api.StashGet[*Order](handler)
		suite.True(ok)
		suite.Same(order, o)
	})

	suite.Run("Drop", func() {
		handler1 := suite.Setup()
		handler2 := miruken.AddHandlers(handler1, api.NewStash(false))
		order := &Order{1, OrderCreated}
		err := api.StashPut(handler2, order)
		suite.Nil(err)
		err = api.StashDrop[*Order](handler2)
		suite.Nil(err)
		_, ok := api.StashGet[*Order](handler2)
		suite.False(ok)
	})

	suite.Run("Cascade", func() {
		handler1 := suite.Setup()
		order := &Order{1, OrderCreated}
		handler2 := miruken.AddHandlers(handler1, api.NewStash(false))
		err := api.StashPut(handler1, order)
		suite.Nil(err)
		o, ok := api.StashGet[*Order](handler2)
		suite.True(ok)
		suite.Same(order, o)
	})

	suite.Run("Hide", func() {
		handler1 := suite.Setup()
		order := &Order{1, OrderCreated}
		handler2 := miruken.AddHandlers(handler1, api.NewStash(false))
		err := api.StashPut(handler1, order)
		suite.Nil(err)
		err = api.StashPut[*Order](handler2, nil)
		suite.Nil(err)
		o, ok := api.StashGet[*Order](handler2)
		suite.True(ok)
		suite.Nil(o)
	})

	suite.Run("Provide", func() {
		handler := suite.Setup()
		order := &Order{1, OrderCreated}
		err := api.StashPut(handler, order)
		suite.Nil(err)
		o, _, err := miruken.Resolve[*Order](handler)
		suite.Nil(err)
		suite.Same(order, o)
	})

	suite.Run("Access", func() {
		handler := suite.Setup()
		order , _, err := miruken.Execute[*Order](handler, &CancelOrder{1})
		suite.Nil(err)
		suite.NotNil(order)
	})
}

func TestStashTestSuite(t *testing.T) {
	suite.Run(t, new(StashTestSuite))
}

