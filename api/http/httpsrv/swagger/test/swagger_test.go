package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/http/httpsrv/swagger"
	"github.com/miruken-go/miruken/api/json/jsonstd"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	CreatePlayer struct {
		Name string
	}

	PlayerResult struct {
		PlayerId int32
	}

	PlayerHandler struct {
		nextId int32
	}
)

func (p *PlayerHandler) CreatePlayer(
	_ *miruken.Handles, create *CreatePlayer,
	ctx miruken.HandleContext,
) *promise.Promise[PlayerResult] {
	id := atomic.AddInt32(&p.nextId,1)
	return promise.Resolve(PlayerResult{id})
}

type SwaggerTestSuite struct {
	suite.Suite
	srv *httptest.Server
}

func (suite *SwaggerTestSuite) Setup(specs ...any) *miruken.Context {
	ctx, _ := miruken.Setup(
		TestFeature, http.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Specs(specs...).
		Context()
	return ctx
}

func (suite *SwaggerTestSuite) SetupTest() {
	ctx, _ := miruken.Setup(
		TestFeature, swagger.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Context()
	suite.srv = httptest.NewServer(httpsrv.NewController(ctx))
}

func (suite *SwaggerTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *SwaggerTestSuite) TestSwagger() {
	suite.Run("Generates Swagger", func() {

	})
}

func TestSwaggerTestSuite(t *testing.T) {
	suite.Run(t, new(SwaggerTestSuite))
}
