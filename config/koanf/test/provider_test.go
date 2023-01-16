package test

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

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
)

type ProviderTestSuite struct {
	suite.Suite
}

func (suite *ProviderTestSuite) TestProvider() {
	suite.Run("Unmarshalls", func() {
		var k = koanf.New(".")
		err := k.Load(file.Provider("../../test/configs/app.json"), json.Parser())
		suite.Nil(err)

		var appConfig AppConfig
		err = k.Unmarshal("", &appConfig)
		suite.Nil(err)

		var dbConfig []DatabaseConfig
		err = k.Unmarshal("database", &dbConfig)
		suite.Nil(err)

		var config map[string]any
		err = k.Unmarshal("", &config)
		suite.Nil(err)

		var env string
		err = k.Unmarshal("env", &env)
		suite.Nil(err)
	})
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
