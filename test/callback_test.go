package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"testing"
)

type CallbackTestSuite struct {
	suite.Suite
	composer miruken.Handler
}

func (suite *CallbackTestSuite) SetupSuite() {
	suite.composer, _ = setup.New().Handler()
}

func (suite *CallbackTestSuite) TestCallback() {
	suite.Run("#Result", func() {
		suite.Run("Single Result", func() {
			var callback miruken.CallbackBase
			callback.AddResult("Hello", suite.composer)
			callback.AddResult("Goodbye", suite.composer)
			suite.Equal(2, callback.ResultCount())
			result, pr := callback.Result(false)
			suite.Nil(pr)
			suite.Equal("Hello", result)
		})

		suite.Run("Multiple Results", func() {
			var callback miruken.CallbackBase
			callback.AddResult("Hello", suite.composer)
			callback.AddResult("Goodbye", suite.composer)
			suite.Equal(2, callback.ResultCount())
			result, pr := callback.Result(true)
			suite.Nil(pr)
			suite.ElementsMatch([]any{"Hello", "Goodbye"}, result)
		})
	})

	suite.Run("Async #Result", func() {
		suite.Run("Single Result", func() {
			var callback miruken.CallbackBase
			callback.AddResult(promise.Resolve("Hello"), suite.composer)
			suite.Equal(1, callback.ResultCount())
			_, pr := callback.Result(false)
			suite.NotNil(pr)
			result, err := pr.Await()
			suite.Nil(err)
			suite.Equal("Hello", result)
		})

		suite.Run("Multiple Results", func() {
			var callback miruken.CallbackBase
			callback.AddResult(promise.Resolve("Hello"), suite.composer)
			callback.AddResult(promise.Resolve("Goodbye"), suite.composer)
			suite.Equal(2, callback.ResultCount())
			_, pr := callback.Result(true)
			suite.NotNil(pr)
			result, err := pr.Await()
			suite.Nil(err)
			suite.ElementsMatch([]any{"Hello", "Goodbye"}, result)
		})
	})

	suite.Run("#ReceiveResult", func() {

	})
}

func TestCallbackTestSuite(t *testing.T) {
	suite.Run(t, new(CallbackTestSuite))
}
