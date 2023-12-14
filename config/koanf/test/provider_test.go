package test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	koanfp "github.com/miruken-go/miruken/config/koanf"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
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

	LoadCustomer    struct{}
	CreateCustomer  struct{}
	CustomerCreated struct{}

	EventStore struct {
		cfg AppConfig
	}

	Gateway struct{}

	Repository struct{}
)

// AppConfig

func (a AppConfig) Validate() error {
	if a.Env == "" {
		return errors.New("environment is missing")
	}
	return nil
}

// EventStore

func (e *EventStore) Constructor(
	_ *struct{ config.Load }, cfg AppConfig,
) {
	e.cfg = cfg
}

func (e *EventStore) Env() string {
	return e.cfg.Env
}

func (e *EventStore) Publish(
	_ *struct{ handles.It }, _ CustomerCreated,
	_ *struct{ config.Load }, cfg map[string]any,
) {
	fmt.Println(cfg["services"].(map[string]any)["eventStoreUrl"])
}

// Gateway

func (g *Gateway) CreateCustomer(
	_ *struct{ handles.It }, _ CreateCustomer,
	_ *struct{ config.Load }, cfg AppConfig,
	ctx miruken.HandleContext,
) error {
	fmt.Println(cfg.Services.CustomerUrl)
	_, err := handles.Command(ctx, CustomerCreated{})
	return err
}

// Repository

func (r *Repository) LoadCustomer(
	_ *struct{ handles.It }, _ LoadCustomer,
	_ *struct {
		config.Load `path:",flat"`
	}, cfg struct {
		Databases []DatabaseConfig `path:"databases"`
	},
) {
	fmt.Printf("%+v\n", cfg.Databases[0])
}

type ProviderTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *ProviderTestSuite) SetupTest() {
	suite.specs = []any{}
}

func (suite *ProviderTestSuite) Setup(specs ...any) (*context.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	var k = koanf.New(".")
	err := k.Load(file.Provider("../../test/configs/appconfig.json"), json.Parser())
	suite.Nil(err)
	return setup.New(config.Feature(koanfp.P(k))).
		Specs(specs...).Context()
}

func (suite *ProviderTestSuite) TestProvider() {
	suite.Run("Load", func() {
		suite.Run("Resolve", func() {
			handler, _ := suite.Setup()
			cfg, _, ok, err := provides.Type[AppConfig](handler, new(config.Load))
			suite.True(ok)
			suite.Nil(err)
			suite.Equal("develop", cfg.Env)
			suite.Len(cfg.Databases, 2)
		})

		suite.Run("Constructor", func() {
			handler, _ := suite.Setup(&EventStore{})
			es, _, ok, err := provides.Type[*EventStore](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Equal("develop", es.Env())
		})

		suite.Run("Method", func() {
			handler, _ := suite.Setup(&Gateway{}, &EventStore{})
			_, err := handles.Command(handler, CreateCustomer{})
			suite.Nil(err)
		})

		suite.Run("Validate", func() {
			handler, _ := suite.Setup()
			_, _, ok, err := provides.Type[AppConfig](handler, &config.Load{Path: "missing"})
			suite.False(ok)
			suite.NotNil(err)
			suite.ErrorContains(err, "config: environment is missing")
		})
	})

	suite.Run("Partial", func() {
		suite.Run("Path", func() {
			type UrlConfig struct {
				EventStoreUrl string
				CustomerUrl   string
			}
			handler, _ := suite.Setup()
			cfg, _, ok, err := provides.Type[UrlConfig](handler, &config.Load{Path: "services"})
			suite.True(ok)
			suite.Nil(err)
			suite.Equal("http://gateway/events", cfg.EventStoreUrl)
			suite.Equal("http://gateway/customer", cfg.CustomerUrl)
		})

		suite.Run("Flat", func() {
			type UrlConfig struct {
				EventStoreUrl string `path:"services.eventStoreUrl"`
				CustomerUrl   string `path:"services.customerUrl"`
			}
			handler, _ := suite.Setup()
			cfg, _, ok, err := provides.Type[UrlConfig](handler, &config.Load{Flat: true})
			suite.True(ok)
			suite.Nil(err)
			suite.Equal("http://gateway/events", cfg.EventStoreUrl)
			suite.Equal("http://gateway/customer", cfg.CustomerUrl)
		})

		suite.Run("Method", func() {
			handler, _ := suite.Setup(&Repository{})
			_, err := handles.Command(handler, LoadCustomer{})
			suite.Nil(err)
		})
	})

	suite.Run("Env", func() {
		_ = os.Setenv("Miruken__Env", "local")
		_ = os.Setenv("Miruken__Services__EventStoreUrl", "http://gateway/events")
		_ = os.Setenv("Miruken__Services__CustomerUrl", "http://gateway/customer")
		_ = os.Setenv("Miruken__Databases__0__Name", "mongo")
		_ = os.Setenv("Miruken__Databases__0__ConnectionString", "mongodb://mongodb0.example.com:27017")
		_ = os.Setenv("Miruken__Databases__0__Timeout", "5h30m40s")
		_ = os.Setenv("Miruken__Databases__1__Name", "sql")
		_ = os.Setenv("Miruken__Databases__1__ConnectionString", "Server=localhost;Database=Customers;User Id=user")
		_ = os.Setenv("Miruken__Databases__1__Timeout", "1h10m20s")
		var k = koanf.New(".")
		err := k.Load(env.Provider("Miruken", "__", nil), nil,
			koanf.WithMergeFunc(koanfp.Merge))
		suite.Nil(err)
		handler, _ := setup.New(config.Feature(koanfp.P(k))).Context()

		suite.Run("Resolve", func() {
			cfg, _, ok, err := provides.Type[AppConfig](handler, &config.Load{Path: "Miruken"})
			suite.True(ok)
			suite.Nil(err)
			suite.Equal("local", cfg.Env)
			suite.Equal("http://gateway/events", cfg.Services.EventStoreUrl)
			suite.Equal("http://gateway/customer", cfg.Services.CustomerUrl)
			suite.Len(cfg.Databases, 2)
		})
	})

	suite.Run("Slices", func() {
		suite.Run("Nothing", func() {
			m := map[string]any{
				"Name": "John",
			}
			s, ok := koanfp.ConvertSlices(m)
			suite.Nil(s)
			suite.False(ok)
		})

		suite.Run("Nothing", func() {
			m := map[string]any{
				"Name": "John",
			}
			s, ok := koanfp.ConvertSlices(m)
			suite.False(ok)
			suite.Nil(s)
		})

		suite.Run("Simple", func() {
			m := map[string]any{
				"0": 12,
				"2": 22,
				"1": 37,
			}
			s, ok := koanfp.ConvertSlices(m)
			suite.True(ok)
			suite.NotNil(s)
			suite.Equal([]any{12, 37, 22}, s)
		})

		suite.Run("Sparse", func() {
			m := map[string]any{
				"10": 42,
				"6":  19,
				"15": 100,
			}
			s, ok := koanfp.ConvertSlices(m)
			sv := reflect.ValueOf(s)
			suite.True(ok)
			suite.NotNil(s)
			suite.Equal(16, sv.Len())
			suite.Equal(42, sv.Index(10).Interface().(int))
			suite.Equal(19, sv.Index(6).Interface().(int))
			suite.Equal(100, sv.Index(15).Interface().(int))
		})

		suite.Run("Nested", func() {
			m := map[string]any{
				"Databases": map[string]any{
					"0": map[string]any{
						"Name": "mongo",
					},
					"1": map[string]any{
						"Name": "sql",
					},
					"2": map[string]any{
						"Name": "postgres",
					},
				},
			}
			s, ok := koanfp.ConvertSlices(m)
			suite.False(ok)
			suite.Nil(s)
			suite.Equal([]any{
				map[string]any{"Name": "mongo"},
				map[string]any{"Name": "sql"},
				map[string]any{"Name": "postgres"},
			}, m["Databases"])
		})
	})
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
