package test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	Launch struct {
		Missile string
	}

	MissileTracked struct {
		LaunchCode string
		Count      int
	}

	MissionControlHandler struct{}

	PresidentHandler struct{}
)

func (m *MissionControlHandler) Launch(
	_ *handles.It, launch Launch,
) *promise.Promise[string] {
	if missile := launch.Missile; missile == "Tomahawk" {
		panic(fmt.Sprintf("launch misfire: %v", missile))
	}
	launchCode := m.launchCode()
	return promise.Resolve(launchCode)
}

func (m *MissionControlHandler) Track(
	_ *handles.It, track *MissileTracked,
) *promise.Promise[struct{}] {
	track.Count++
	if track.Count == 2 {
		return promise.Empty()
	}
	return nil
}

func (p *PresidentHandler) Track(
	_ *handles.It, track *MissileTracked,
) *promise.Promise[struct{}] {
	track.Count++
	if track.Count == 2 {
		return promise.Empty()
	}
	return nil
}

func (m *MissionControlHandler) launchCode() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

type MessageTestSuite struct {
	suite.Suite
}

func (suite *MessageTestSuite) Setup() *context.Context {
	ctx, _ := setup.New(
		TestFeature,
		api.Feature(),
	).Context()
	return ctx
}

func (suite *MessageTestSuite) TestMessage() {
	suite.Run("Success", func() {
		red := api.Success("red")
		blue := api.Success("blue")
		result := either.FlatMap(red, func(r1 string) either.Monad[error, string] {
			return either.FlatMap(blue, func(r2 string) either.Monad[error, string] {
				return api.Success(fmt.Sprintf("%v %v", r1, r2))
			})
		})
		either.Match(result,
			func(error) { suite.Fail("unexpected") },
			func(s string) { suite.Equal("red blue", s) })
	})

	suite.Run("Failure", func() {
		red := api.Success("red")
		blue := api.Success("blue")
		result := either.FlatMap(red, func(r1 string) either.Monad[error, string] {
			return either.FlatMap(blue, func(r2 string) either.Monad[error, string] {
				return api.Failure(errors.New("broken"))
			})
		})
		either.Match(result,
			func(err error) { suite.Equal("broken", err.Error()) },
			func(string) { suite.Fail("unexpected") })
	})

	suite.Run("Post", func() {
		handler := suite.Setup()
		p, err := api.Post(handler, Launch{Missile: "Patriot"})
		suite.Nil(err)
		suite.NotNil(p)
		_, err = p.Await()
		suite.Nil(err)
	})

	suite.Run("Send", func() {
		handler := suite.Setup()
		launch := Launch{Missile: "Patriot"}
		_, pc, err := api.Send[string](handler, launch)
		suite.Nil(err)
		suite.NotNil(pc)
		code, err := pc.Await()
		suite.Nil(err)
		suite.NotEmpty(code)
	})

	suite.Run("Send Panic", func() {
		handler := suite.Setup()
		launch := Launch{Missile: "Tomahawk"}
		_, _, err := api.Send[string](handler, launch)
		suite.NotNil(err)
		suite.Equal("send: panic: launch misfire: Tomahawk", err.Error())
	})

	suite.Run("Publish", func() {
		handler := suite.Setup()
		launch := Launch{Missile: "Patriot"}
		_, pc, err := api.Send[string](handler, launch)
		suite.Nil(err)
		code, err := pc.Await()
		suite.Nil(err)
		tracked := &MissileTracked{LaunchCode: code}
		pt, err := api.Publish(handler, tracked)
		suite.Nil(err)
		suite.NotNil(pt)
		_, err = pt.Await()
		suite.Equal(2, tracked.Count)
	})
}

func TestMessageTestSuite(t *testing.T) {
	suite.Run(t, new(MessageTestSuite))
}
