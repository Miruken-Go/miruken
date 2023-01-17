package test

import (
	"fmt"
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

type (
	AppConfig struct {
		Env       string
		Databases []DatabaseConfig
		Services  ServiceConfig
	}

	ServiceConfig struct {
		EventStoreUrl string
		CustomerUrl   string
	}

	DatabaseConfig struct {
		Name             string
		ConnectionString string
		Timeout          time.Duration
	}

	LoadCustomer struct {}
	CreateCustomer struct {}
	CustomerCreated struct {}

	EventStore struct {
		cfg AppConfig
	}

	Gateway struct {}

	Repository struct {}
)

// EventStore

func (e *EventStore) Constructor(
	_*struct{config.Load}, cfg AppConfig,
) {
	e.cfg = cfg
}

func (e *EventStore) Env() string {
	return e.cfg.Env
}

func (e *EventStore) Publish(
	_*struct{miruken.Handles}, _ CustomerCreated,
	_*struct{config.Load}, cfg map[string]any,
) {
	fmt.Println(cfg["services"].(map[string]any)["eventStoreUrl"])
}

// Gateway

func (g *Gateway) CreateCustomer(
	_*struct{miruken.Handles}, _ CreateCustomer,
	_*struct{config.Load}, cfg AppConfig,
	ctx miruken.HandleContext,
) error {
	fmt.Println(cfg.Services.CustomerUrl)
	_, err := miruken.Command(ctx.Composer(), CustomerCreated{})
	return err
}

// Repository

func (r *Repository) LoadCustomer(
	_*struct{miruken.Handles}, _ LoadCustomer,
	_*struct{config.Load `path:",flat"`}, cfg struct {
		Databases []DatabaseConfig `path:"databases"`
	},
) {
	fmt.Printf("%+v\n", cfg.Databases[0])
}

type ProviderTestSuite struct {
	suite.Suite
}

func (suite *ProviderTestSuite) TestProvider() {
	var k = koanf.New(".")
	err := k.Load(file.Provider("../../test/configs/app.json"), json.Parser())
	suite.Nil(err)

	suite.Run("Load", func() {
		suite.Run("Resolve", func() {
			handler, _ := miruken.Setup(config.Feature(koanfp.P(k)))
			cfg, _, err := miruken.Resolve[AppConfig](handler, new(config.Load))
			suite.Nil(err)
			suite.Equal("develop", cfg.Env)
			suite.Len(cfg.Databases, 1)
		})

		suite.Run("Constructor", func() {
			handler, _ := miruken.Setup(
				config.Feature(koanfp.P(k)),
				miruken.HandlerSpecs(&EventStore{}))
			es, _, err := miruken.Resolve[*EventStore](handler)
			suite.Nil(err)
			suite.Equal("develop", es.Env())
		})

		suite.Run("Method", func() {
			handler, _ := miruken.Setup(
				config.Feature(koanfp.P(k)),
				miruken.HandlerSpecs(&Gateway{}, &EventStore{}))
			_, err := miruken.Command(handler, CreateCustomer{})
			suite.Nil(err)
		})
	})

	suite.Run("Partial", func() {
		suite.Run("Path", func() {
			type UrlConfig struct {
				EventStoreUrl string
				CustomerUrl   string
			}
			handler, _ := miruken.Setup(config.Feature(koanfp.P(k)))
			cfg, _, err := miruken.Resolve[UrlConfig](handler, &config.Load{Path: "services"})
			suite.Nil(err)
			suite.Equal("http://gateway/events", cfg.EventStoreUrl)
			suite.Equal("http://gateway/customer", cfg.CustomerUrl)
		})

		suite.Run("Flat", func() {
			type UrlConfig struct {
				EventStoreUrl string `path:"services.eventStoreUrl"`
				CustomerUrl   string `path:"services.customerUrl"`
			}
			handler, _ := miruken.Setup(config.Feature(koanfp.P(k)))
			cfg, _, err := miruken.Resolve[UrlConfig](handler, &config.Load{Flat: true})
			suite.Nil(err)
			suite.Equal("http://gateway/events", cfg.EventStoreUrl)
			suite.Equal("http://gateway/customer", cfg.CustomerUrl)
		})

		suite.Run("Method", func() {
			handler, _ := miruken.Setup(
				config.Feature(koanfp.P(k)),
				miruken.HandlerSpecs(&Repository{}))
			_, err := miruken.Command(handler, LoadCustomer{})
			suite.Nil(err)
		})
	})
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
