package test

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/http/httpsrv/openapi"
	"github.com/miruken-go/miruken/api/json/stdjson"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	Address struct {
		Line  string
		City  string
		State string
		Zip   string
	}

	PlayerData struct {
		Id        int32
		Name      string
		BirthDate time.Time
		Address   Address
		Version   int32
	}

	CreatePlayer struct {
		Name      string
		BirthDate time.Time
		Address   Address
	}

	UpdatePlayer struct {
		Id        int32
		Name      string
		BirthDate time.Time
		Address   Address
	}

	PlayerResult struct {
		Id      int32
		Version int32
	}

	PlayerHandler struct {
		nextId int32
		store  map[int32]*PlayerData
	}
)

func (p *PlayerHandler) Constructor() {
	p.store = make(map[int32]*PlayerData)
}

func (p *PlayerHandler) CreatePlayer(
	_ *handles.It, create CreatePlayer,
) *promise.Promise[PlayerResult] {
	id     := atomic.AddInt32(&p.nextId,1)
	player := PlayerData{
		Id:        id,
		Name:      create.Name,
		BirthDate: create.BirthDate,
		Address:   create.Address,
		Version:   1,
	}
	p.store[id] = &player
	return promise.Resolve(PlayerResult{id, player.Version})
}

func (p *PlayerHandler) UpdatePlayer(
	_ *handles.It, update UpdatePlayer,
) *promise.Promise[PlayerResult] {
	if player, ok := p.store[update.Id]; !ok {
		nf := fmt.Errorf("player with id %v not found", update.Id)
		return promise.Reject[PlayerResult](nf)
	} else {
		player.Version++
		return promise.Resolve(PlayerResult{player.Id, player.Version})
	}
}

type OpenApiTestSuite struct {
	suite.Suite
	openapi *openapi.Installer
	ctx *context.Context
	srv *httptest.Server
}

func (suite *OpenApiTestSuite) Setup(specs ...any) miruken.Handler {
	handler, _ := setup.New(
		TestFeature, http.Feature(), stdjson.Feature()).
		Specs(&api.GoPolymorphism{}).
		Specs(specs...).
		Handler()
	return handler
}

func (suite *OpenApiTestSuite) SetupTest() {
	suite.openapi = openapi.Feature(openapi3.T{
		Info: &openapi3.Info{
			Title:       "Team Api",
			Description: "REST Api used for managing Teams",
			License: &openapi3.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
			Contact: &openapi3.Contact{
				URL: "https://github.com/craig/team-microservice",
			},
		},
		Servers: openapi3.Servers{
			&openapi3.Server{
				Description: "Local development",
				URL:         "http://127.0.0.1:9234",
			},
		},
	})
	handler, _ := setup.New(
		TestFeature, stdjson.Feature(), suite.openapi).
		Specs(&api.GoPolymorphism{}).
		Handler()
	suite.ctx = context.New(handler)
	suite.srv = httptest.NewServer(httpsrv.Api(suite.ctx))
}

func (suite *OpenApiTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
	suite.ctx.End(nil)
}

func (suite *OpenApiTestSuite) TestOpenApi() {
	suite.Run("Generates OpenApi", func() {
		docs := suite.openapi.Docs()
		suite.Len(docs, 1)
	})
}

func TestOpenApiTestSuite(t *testing.T) {
	suite.Run(t, new(OpenApiTestSuite))
}
