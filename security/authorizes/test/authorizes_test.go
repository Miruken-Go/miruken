package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
)

type AuthorizesTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *AuthorizesTestSuite) SetupTest() {
	suite.specs = []any{
	}
}

func (suite *AuthorizesTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup().Specs(suite.specs...).Handler()
}

func (suite *AuthorizesTestSuite) TestAuthorizes() {
}