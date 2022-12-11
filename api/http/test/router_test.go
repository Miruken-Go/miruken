package test

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/validate"
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

	TeamCreated struct {
		Team TeamData
	}

	GetTeamNotifications struct {}

	TeamApiHandler struct {
		nextId int32
	}

	TeamApiConsumer struct {
		notifications []any
	}
)

// TeamApiHandler

func (t *TeamApiHandler) MustHaveTeamName(
	validates *validate.Validates, create *CreateTeam,
) {
	outcome := validates.Outcome()

	if len(create.Name) == 0 {
		outcome.AddError("Name", errors.New(`"Name" is required`))
	}
}

func (t *TeamApiHandler) CreateTeam(
	_*miruken.Handles, create *CreateTeam,
	ctx miruken.HandleContext,
) *promise.Promise[*TeamData] {
	id := atomic.AddInt32(&t.nextId,1)
	team := &TeamData{id,create.Name, create.Players}
	_, _ = api.Publish(ctx.Composer(), &TeamCreated{Team: *team})
	return promise.Resolve(team)
}

func (t *TeamApiHandler) NewCreateTeam(
	_*struct{
		miruken.Creates `key:"test.CreateTeam"`
	  }, _ *miruken.Creates,
) *CreateTeam {
	return &CreateTeam{}
}

func (t *TeamApiHandler) NewTeamCreated(
	_*struct{
		miruken.Creates `key:"test.TeamCreated"`
	  }, _ *miruken.Creates,
) *TeamCreated {
	return &TeamCreated{}
}

func (t *TeamApiHandler) NewGetTeamNotifications(
	_*struct{
		miruken.Creates `key:"test.GetTeamNotifications"`
	}, _ *miruken.Creates,
) *GetTeamNotifications {
	return &GetTeamNotifications{}
}

func (t *TeamApiHandler) NewTeam(
	_*struct{
		miruken.Creates `key:"test.TeamData"`
	  }, _ *miruken.Creates,
) *TeamData {
	return &TeamData{}
}


// TeamApiConsumer

func (t *TeamApiConsumer) TeamCreated(
	_*miruken.Handles, created *TeamCreated,
) {
	t.notifications = append(t.notifications, created)
}

func (t *TeamApiConsumer) TeamNotifications(
	_*miruken.Handles, get *GetTeamNotifications,
) []any {
	return t.notifications
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
		http.ServerFeature(),
		miruken.HandlerSpecs(&json.GoTypeFieldMapper{}))
	ctrl := &http.Controller{Context: miruken.NewContext(handler)}
	suite.srv = httptest.NewServer(ctrl)
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *RouterTestSuite) TestRouter() {
	suite.Run("Route", func() {
		suite.Run("Send", func() {
			handler := suite.Setup()
			create := api.RouteTo(CreateTeam{Name: "Tottenham"}, suite.srv.URL)
			_, pp, err := api.Send[*TeamData](handler, create)
			suite.Nil(err)
			suite.NotNil(pp)
			team, err := pp.Await()
			suite.Nil(err)
			suite.Equal(TeamData{1, "Tottenham", nil}, *team)

			get := api.RouteTo(GetTeamNotifications{}, suite.srv.URL)
			events, pe, err := api.Send[[]any](handler, get)
			suite.Nil(err)
			suite.NotNil(pe)
			events, err = pe.Await()
			suite.Nil(err)
			suite.NotNil(events)
			created := &TeamCreated{TeamData{1, "Tottenham", nil}}
			suite.Contains(events, created)
		})

		suite.Run("Publish", func() {
			handler := suite.Setup()
			created := TeamCreated{TeamData{8, "Liverpool", nil}}
			notify  := api.RouteTo(created, suite.srv.URL)
			pv, err := api.Publish(handler, notify)
			suite.Nil(err)
			suite.NotNil(pv)
			_, err = pv.Await()
			suite.Nil(err)

			get := api.RouteTo(GetTeamNotifications{}, suite.srv.URL)
			events, pe, err := api.Send[[]any](handler, get)
			suite.Nil(err)
			suite.NotNil(pe)
			events, err = pe.Await()
			suite.Nil(err)
			suite.NotNil(events)
			ev := &TeamCreated{TeamData{8, "Liverpool", nil}}
			suite.Contains(events, ev)
		})

		suite.Run("ValidationError", func() {
			handler := suite.Setup()
			create := api.RouteTo(CreateTeam{}, suite.srv.URL)
			_, pp, err := api.Send[*TeamData](handler, create)
			suite.Nil(err)
			suite.NotNil(pp)
			_, err = pp.Await()
			suite.NotNil(err)
		})
	})
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
