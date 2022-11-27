package test

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/json"
	"github.com/stretchr/testify/suite"
	http2 "net/http"
	"net/http/httptest"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type RouterTestSuite struct {
	suite.Suite
	srv *httptest.Server
}

func (suite *RouterTestSuite) Setup(specs ... any) (miruken.Handler, error) {
	return miruken.Setup(
		http.Feature(),
		miruken.HandlerSpecs(&json.GoTypeFieldMapper{}),
		miruken.HandlerSpecs(specs...))
}

func (suite *RouterTestSuite) SetupTest() {
	handler := http2.HandlerFunc(func(w http2.ResponseWriter, r *http2.Request) {
		w.WriteHeader(http2.StatusOK)
	})
	suite.srv = httptest.NewServer(handler)
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *RouterTestSuite) TestRouter() {
	suite.Run("Route", func() {
		fmt.Println(suite.srv.URL)
	})
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
