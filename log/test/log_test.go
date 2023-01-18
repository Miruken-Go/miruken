package test

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/log"
	"github.com/miruken-go/miruken/promise"
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
	_*miruken.Handles, cmd Command,
	logger logr.Logger,
) Command {
	var level = int(cmd)
	logger.V(level).Info("executed command", "level", level)
	return cmd+1
}

func (s *Service) LongCommand(
	_*miruken.Handles, cmd LongCommand,
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
	suite.Run("Provides", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.New(suite.T())),
		)
		logger, _, err:= miruken.Resolve[logr.Logger](handler)
		suite.Nil(err)
		logger.Info("Hello")
	})

	suite.Run("Verbosity", func() {
		handler, _ := miruken.Setup(
			log.Feature(
				testr.NewWithOptions(suite.T(), testr.Options{Verbosity: 1}),
			),
		)
		logger, _, err:= miruken.Resolve[logr.Logger](handler)
		suite.Nil(err)
		logger.V(1).Info("World")
	})

	suite.Run("CtorDependency", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.New(suite.T())),
			miruken.Specs(&Service{}),
		)
		svc, _, err := miruken.Resolve[*Service](handler)
		suite.Nil(err)
		svc.Run()
	})

	suite.Run("MethodDependency", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.NewWithOptions(suite.T(), testr.Options{
				LogTimestamp: true,
				Verbosity:    1,
			})),
			miruken.Specs(&Service{}),
		)
		next, _, err := miruken.Execute[Command](handler, Command(1))
		suite.Nil(err)
		suite.Equal(Command(2), next)
		next, _, err = miruken.Execute[Command](handler, next)
		suite.Nil(err)
		suite.Equal(Command(3), next)
	})

	suite.Run("MethodDependencyAsync", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.NewWithOptions(suite.T(), testr.Options{
				LogTimestamp: true,
				Verbosity:    1,
			})),
			miruken.Specs(&Service{}),
		)
		next, np, err := miruken.Execute[LongCommand](handler, LongCommand(8))
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
			}), log.Verbosity(2)),
			miruken.Specs(&Service{}),
		)
		next, _, err := miruken.Execute[Command](handler, Command(2))
		suite.Nil(err)
		suite.Equal(Command(3), next)
	})
}

func TestLogTestSuite(t *testing.T) {
	suite.Run(t, new(LogTestSuite))
}

