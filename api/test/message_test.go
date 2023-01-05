package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"os/exec"
	"strings"
	"testing"
)

type (
	Launch struct {
		Missile  string
	}

	MissileTracked struct {
		LaunchCode string
		Count      int
	}

	MissionControlHandler struct{}

	PresidentHandler struct {}
)

func (m *MissionControlHandler) Launch(
	_ *miruken.Handles, launch Launch,
) *promise.Promise[string] {
	if missile := launch.Missile; missile == "Tomahawk" {
		panic(fmt.Sprintf("launch misfire: %v", missile))
	}
	launchCode := m.launchCode()
	return promise.Resolve(launchCode)
}

func (m *MissionControlHandler) Track(
	_ *miruken.Handles, track *MissileTracked,
) *promise.Promise[miruken.Void] {
	track.Count++
	if track.Count == 2 {
		return promise.Resolve(miruken.Void{})
	}
	return nil
}

func (p *PresidentHandler) Track(
	_ *miruken.Handles, track *MissileTracked,
) *promise.Promise[miruken.Void] {
	track.Count++
	if track.Count == 2 {
		return promise.Resolve(miruken.Void{})
	}
	return nil
}

func (m *MissionControlHandler) launchCode() string {
	if newUUID, err := exec.Command("uuidgen").Output(); err != nil {
		panic(err)
	} else {
		return strings.TrimSuffix(string(newUUID), "\n")
	}
}

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
	suite.Run("Success", func() {
		red    := api.Success("red")
		blue   := api.Success("blue")
		result := either.FlatMap(red, func(r1 string) either.Either[error, string] {
			return either.FlatMap(blue, func(r2 string) either.Either[error, string] {
				return api.Success(fmt.Sprintf("%v %v", r1, r2))
			})
		})
		either.Match(result,
			func(error) { suite.Fail("unexpected") },
			func(s string) { suite.Equal("red blue", s) })
	})

	suite.Run("Failure", func() {
		red    := api.Success("red")
		blue   := api.Success("blue")
		result := either.FlatMap(red, func(r1 string) either.Either[error, string] {
			return either.FlatMap(blue, func(r2 string) either.Either[error, string] {
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
		launch  := Launch{Missile: "Patriot"}
		code, pc, err := api.Send[string](handler, launch)
		suite.Nil(err)
		suite.Empty(code)
		suite.NotNil(pc)
		code, err = pc.Await()
		suite.Nil(err)
		suite.NotEmpty(code)
	})

	suite.Run("Send Panic", func() {
		handler := suite.Setup()
		launch  := Launch{Missile: "Tomahawk"}
		_, _, err := api.Send[string](handler, launch)
		suite.NotNil(err)
		suite.Equal("send: panic: launch misfire: Tomahawk", err.Error())
	})

	suite.Run("Publish", func() {
		handler := suite.Setup()
		launch  := Launch{Missile: "Patriot"}
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
