package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/json"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	PlayerData struct {
		Id   int32
		Name string
	}

	TeamData struct {
		Id      int32
		Name    string
		Players []PlayerData
	}

	CreateTeam struct {
		Name    string
		Players []PlayerData
	}

	TeamApiHandler struct {
		nextId int32
	}
)

// TeamApiHandler

func (t *TeamApiHandler) CreateTeam(
	_*miruken.Handles, create *CreateTeam,
) *TeamData {
	id := atomic.AddInt32(&t.nextId,1)
	return &TeamData{id,create.Name, create.Players}
}

func (t *TeamApiHandler) NewCreateTeam(
	_*struct{
		miruken.Creates `key:"test.CreateTeam"`
	  }, _ *miruken.Creates,
) *CreateTeam {
	return &CreateTeam{}
}

func (t *TeamApiHandler) NewTeam(
	_*struct{
		miruken.Creates `key:"test.TeamData"`
	  }, _ *miruken.Creates,
) *TeamData {
	return &TeamData{}
}

type RouterTestSuite struct {
	suite.Suite
	srv *httptest.Server
}

func (suite *RouterTestSuite) Setup() *miruken.Context {
	handler, _ := miruken.Setup(
		TestFeature,
		http.Feature(),
		miruken.HandlerSpecs(&json.GoTypeFieldMapper{}))
	return miruken.NewContext(handler)
}

func (suite *RouterTestSuite) SetupTest() {
	handler, _ := miruken.Setup(
		TestFeature,
		http.Feature(),
		miruken.HandlerSpecs(&json.GoTypeFieldMapper{}))
	controller := &http.Controller{}
	controller.SetContext(miruken.NewContext(handler))
	suite.srv = httptest.NewServer(controller)
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *RouterTestSuite) TestRouter() {
	suite.Run("Route", func() {
		handler := suite.Setup()
		create  := api.RouteTo(CreateTeam{Name: "Tottenham"}, suite.srv.URL)
		_, pp, err := api.Send[*TeamData](handler, create)
		suite.Nil(err)
		suite.NotNil(pp)
		team, err := pp.Await()
		suite.Nil(err)
		suite.Equal(TeamData{1, "Tottenham", nil}, *team)
	})
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
