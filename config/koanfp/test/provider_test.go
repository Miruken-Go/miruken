package test

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	"github.com/miruken-go/miruken/config/koanfp"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	AppConfig struct {
		Env      string
		Database []DatabaseConfig
	}

	DatabaseConfig struct {
		Name             string
		ConnectionString string
		Timeout          time.Duration
	}

	Event struct {
		id   int
		data []byte
	}

	EventStore struct {
		cfg AppConfig
	}
)

func (e *EventStore) Constructor(
	_*struct{config.Load}, cfg AppConfig,
) {
	e.cfg = cfg
}

func (e *EventStore) Env() string {
	return e.cfg.Env
}

type ProviderTestSuite struct {
	suite.Suite
}

func (suite *ProviderTestSuite) TestProvider() {
	suite.Run("Resolve", func() {
		var k = koanf.New(".")
		err := k.Load(file.Provider("../../test/configs/app.json"), json.Parser())
		suite.Nil(err)

		suite.Run("Resolve", func() {
			handler, _ := miruken.Setup(config.Feature(koanfp.P(k)))
			cfg, _, err := miruken.Resolve[AppConfig](handler, config.Load{})
			suite.Nil(err)
			suite.Equal("develop", cfg.Env)
		})

		suite.Run("Constructor", func() {
			handler, _ := miruken.Setup(
				config.Feature(koanfp.P(k)),
				miruken.HandlerSpecs(&EventStore{}))
			es, _, err := miruken.Resolve[*EventStore](handler)
			suite.Nil(err)
			suite.Equal("develop", es.Env())
		})
	})
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
