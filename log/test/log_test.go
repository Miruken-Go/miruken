package test

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/log"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type (
	Service struct {
		logger logr.Logger
	}

	Command int
	LongCommand int64
)

func (s *Service) Constructor(
	logger logr.Logger,
) {
	s.logger = logger
}

func (s *Service) Run() {
	s.logger.WithName("Run").Info("starting service")
}

func (s *Service) Command(
	_ *handles.It, cmd Command,
	logger logr.Logger,
) Command {
	var level = int(cmd)
	logger.V(level).Info("executed command", "level", level)
	return cmd+1
}

func (s *Service) LongCommand(
	_ *handles.It, cmd LongCommand,
	logger logr.Logger,
) *promise.Promise[LongCommand] {
	duration := time.Duration(cmd) * time.Millisecond
	logger.Info("executed long command", "duration", duration)
	_, _ = promise.Delay(duration).Await()
	return promise.Resolve(cmd+1)
}

type LogTestSuite struct {
	suite.Suite
}

func (suite *LogTestSuite) TestLogging() {
	suite.Run("Build", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.New(suite.T())),
		).Handler()
		logger, _, err:= provides.Type[logr.Logger](handler)
		suite.Nil(err)
		logger.Info("Hello")
	})

	suite.Run("verbosity", func() {
		handler, _ := miruken.Setup(
			log.Feature(
				testr.NewWithOptions(suite.T(), testr.Options{Verbosity: 1}),
			),
		).Handler()
		logger, _, err:= provides.Type[logr.Logger](handler)
		suite.Nil(err)
		logger.V(1).Info("World")
	})

	suite.Run("CtorDependency", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.New(suite.T()))).
			Specs(&Service{}).
			Handler()
		svc, _, err := provides.Type[*Service](handler)
		suite.Nil(err)
		svc.Run()
	})

	suite.Run("MethodDependency", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.NewWithOptions(suite.T(), testr.Options{
				LogTimestamp: true,
				Verbosity:    1,
			}))).
			Specs(&Service{}).
			Handler()
		next, _, err := handles.Request[Command](handler, Command(1))
		suite.Nil(err)
		suite.Equal(Command(2), next)
		next, _, err = handles.Request[Command](handler, next)
		suite.Nil(err)
		suite.Equal(Command(3), next)
	})

	suite.Run("MethodDependencyAsync", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.NewWithOptions(suite.T(), testr.Options{
				LogTimestamp: true,
				Verbosity:    1,
			}))).
			Specs(&Service{}).
			Handler()
		next, np, err := handles.Request[LongCommand](handler, LongCommand(8))
		suite.Nil(err)
		suite.NotNil(np)
		next, err = np.Await()
		suite.Nil(err)
		suite.Equal(LongCommand(9), next)
	})

	suite.Run("Suppressed", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.NewWithOptions(suite.T(), testr.Options{
				LogTimestamp: true,
				Verbosity:    1,
			}), log.Verbosity(2))).
			Specs(&Service{}).
			Handler()
		next, _, err := handles.Request[Command](handler, Command(2))
		suite.Nil(err)
		suite.Equal(Command(3), next)
	})
}

func TestLogTestSuite(t *testing.T) {
	suite.Run(t, new(LogTestSuite))
}

