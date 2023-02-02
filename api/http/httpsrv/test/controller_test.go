package test

import (
	json2 "encoding/json"
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/json/jsonstd"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/validates"
	"github.com/stretchr/testify/suite"
	"io"
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

	BadFormatter struct {}
)

// TeamApiHandler

func (t *TeamApiHandler) MustHaveTeamName(
	v *validates.It, create *CreateTeam,
) {
	outcome := v.Outcome()

	if len(create.Name) == 0 {
		outcome.AddError("Name", errors.New(`"Name" is required`))
	}
}

func (t *TeamApiHandler) CreateTeam(
	_ *handles.It, create *CreateTeam,
	ctx miruken.HandleContext,
) *promise.Promise[*TeamData] {
	id := atomic.AddInt32(&t.nextId,1)
	team := &TeamData{id,create.Name, create.Players}
	_, _ = api.Publish(ctx.Composer(), &TeamCreated{Team: *team})
	return promise.Resolve(team)
}

func (t *TeamApiHandler) New(
	_*struct{
		ct  creates.It `key:"test.CreateTeam"`
		tc  creates.It `key:"*test.TeamCreated"`
	    gtn creates.It `key:"test.GetTeamNotifications"`
		td  creates.It `key:"*test.TeamData"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "test.CreateTeam":
		return new(CreateTeam)
	case "*test.TeamCreated":
		return new(TeamCreated)
	case "test.GetTeamNotifications":
		return new(GetTeamNotifications)
	case "*test.TeamData":
		return new(TeamData)
	}
	return nil
}

// TeamApiConsumer

func (t *TeamApiConsumer) TeamCreated(
	_ *handles.It, created *TeamCreated,
) {
	t.notifications = append(t.notifications, created)
}

func (t *TeamApiConsumer) TeamNotifications(
	_ *handles.It, _ *GetTeamNotifications,
) []any {
	return t.notifications
}


// BadFormatter

func (f *BadFormatter) Bad(
	_*struct{
		miruken.Maps
		miruken.Format `to:"bad"`
	  }, msg api.Message,
	maps *miruken.Maps,
) (io.Writer, error) {
	if writer, ok := maps.Target().(*io.Writer); ok && !miruken.IsNil(writer) {
		enc := json2.NewEncoder(*writer)
		err := enc.Encode(msg.Payload)
		return *writer, err
	}
	return nil, nil
}

type ControllerTestSuite struct {
	suite.Suite
	srv *httptest.Server
}

func (suite *ControllerTestSuite) Setup(specs ...any) *context.Context {
	handler, _ := miruken.Setup(
		TestFeature, http.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Specs(specs...).
		Handler()
	return context.New(handler)
}

func (suite *ControllerTestSuite) SetupTest() {
	handler, _ := miruken.Setup(
		TestFeature, httpsrv.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Handler()
	suite.srv = httptest.NewServer(httpsrv.Api(context.New(handler)))
}

func (suite *ControllerTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *ControllerTestSuite) TestController() {
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
			created := &TeamCreated{TeamData{8, "Liverpool", nil}}
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
			create  := api.RouteTo(CreateTeam{}, suite.srv.URL)
			_, pp, err := api.Send[*TeamData](handler, create)
			suite.Nil(err)
			suite.NotNil(pp)
			_, err = pp.Await()
			var outcome *validates.Outcome
			suite.ErrorAs(err, &outcome)
			suite.Equal(`Name: "Name" is required`, outcome.Error())
			suite.Equal([]string{"Name"}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New(`"Name" is required`)},
				outcome.FieldErrors("Name"))
		})

		suite.Run("UnknownFormat", func() {
			handler := miruken.BuildUp(
				suite.Setup(&BadFormatter{}),
				http.Format("bad"))
			create  := api.RouteTo(CreateTeam{}, suite.srv.URL)
			_, pp, err := api.Send[*TeamData](handler, create)
			suite.Nil(err)
			suite.NotNil(pp)
			_, err = pp.Await()
			suite.ErrorContains(err, "415 Unsupported Media Type")
			var nh *miruken.NotHandledError
			suite.ErrorAs(err, &nh)
		})
	})
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
