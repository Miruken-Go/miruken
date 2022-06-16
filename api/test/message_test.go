package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MessageTestSuite struct {
	suite.Suite
}

func (suite *MessageTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		api.Feature(),
	)
	return handler
}

func (suite *MessageTestSuite) TestMessage() {
	suite.Run("Post", func() {

	})

	suite.Run("Send", func() {

	})

	suite.Run("Publish", func() {

	})
}

func TestMessageTestSuite(t *testing.T) {
	suite.Run(t, new(MessageTestSuite))
}
