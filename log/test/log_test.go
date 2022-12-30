package test

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/log"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	Service struct {
		logger logr.Logger
	}

	Command int
)

func (s *Service) Constructor(
	logger logr.Logger,
) {
	s.logger = logger
}

func (s *Service) Run() {
	s.logger.Info("starting service")
}

func (s *Service) Handle(
	_*miruken.Handles, cmd Command,
	logger logr.Logger,
) Command {
	logger.Info("command handled", "cmd", cmd)
	return cmd+1
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
			miruken.HandlerSpecs(&Service{}),
		)
		svc, _, err := miruken.Resolve[*Service](handler)
		suite.Nil(err)
		svc.Run()
	})

	suite.Run("MethodDependency", func() {
		handler, _ := miruken.Setup(
			log.Feature(testr.New(suite.T())),
			miruken.HandlerSpecs(&Service{}),
		)
		next, _, err := miruken.Execute[Command](handler, Command(2))
		suite.Nil(err)
		suite.Equal(Command(3), next)
	})
}

func TestLogTestSuite(t *testing.T) {
	suite.Run(t, new(LogTestSuite))
}

