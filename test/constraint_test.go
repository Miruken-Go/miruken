package test

import (
	"github.com/stretchr/testify/suite"
	"miruken.com/miruken"
	"reflect"
	"testing"
)

type Person interface {
	FirstName() string
	LastName()  string
}

type PersonStruct struct {
	firstName string
	lastName  string
}

func (p *PersonStruct) FirstName() string {
	return p.firstName
}

func (p *PersonStruct) LastName() string {
	return p.lastName
}

type Doctor struct {
	miruken.Metadata
}

func (d *Doctor) Init() error {
	return d.InitWithMetadata(miruken.KeyValues{
		"Job": "Doctor",
	})
}

type Programmer struct{
	miruken.Qualifier
}

func (p *Programmer) Require(metadata *miruken.BindingMetadata) {
	p.RequireQualifier(p, metadata)
}

func (p *Programmer) Matches(metadata *miruken.BindingMetadata) bool {
	return p.MatchesQualifier(p, metadata)
}

type Hospital struct {
	doctor     Person
	programmer Person
}

func (h *Hospital) Constructor(
	doctor     *struct{ Doctor;     Value Person },
	programmer *struct{ Programmer; Value Person },
) {
	h.doctor     = doctor.Value
	h.programmer = programmer.Value
}

func (h *Hospital) Doctor() Person {
	return h.doctor
}

func (h *Hospital) Programmer() Person {
	return h.programmer
}

type PersonProvider struct{}


func (p *PersonProvider) Doctor(
	_ *struct{
		miruken.Provides
		miruken.Singleton
		Doctor
      },
) Person {
	return &PersonStruct{"Jack", "Zigler"}
}

func (p *PersonProvider) Programmer(
	_ *struct{
		miruken.Provides
		miruken.Singleton
		Programmer
      },
) Person {
	return &PersonStruct{"Paul", "Allen"}
}

type AppSettings interface {
	ServerUrl() string
}

type LocalSettings struct{}

func (l *LocalSettings) Constructor(
	_ *struct{
		miruken.Provides
		miruken.Singleton
		miruken.Named `name:"local"`
      },
) {
}

func (l *LocalSettings) ServerUrl() string {
	return "http://localhost/Server"
}

type RemoteSettings struct{}

func (r *RemoteSettings) Constructor(
	_ *struct{
		miruken.Provides
		miruken.Singleton
		miruken.Named `name:"remote"`
      },
) {
}

func (r *RemoteSettings) ServerUrl() string {
	return "https://remote/Server"
}

type Client struct {
	local  AppSettings
	remote AppSettings
}

func (c *Client) Constructor(
	local *struct{
		miruken.Named `name:"local"`
		Value AppSettings
	  },
	remote *struct{
		miruken.Named `name:"remote"`
		Value AppSettings
	  },
) {
	c.local  = local.Value
	c.remote = remote.Value
}

func (c *Client) Local() AppSettings {
	return c.local
}

func (c *Client) Remote() AppSettings {
	return c.remote
}

type ConstraintTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *ConstraintTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		reflect.TypeOf((*PersonProvider)(nil)),
		reflect.TypeOf((*LocalSettings)(nil)),
		reflect.TypeOf((*RemoteSettings)(nil)),
		reflect.TypeOf((*Hospital)(nil)),
		reflect.TypeOf((*Client)(nil)),
	}
	suite.HandleTypes = handleTypes
}

func (suite *ConstraintTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *ConstraintTestSuite) InferenceRootWith(
	handlerTypes ... reflect.Type) miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(handlerTypes...))
}

func (suite *ConstraintTestSuite) TestConstraints() {
	suite.Run("No Constraints", func () {
		suite.Run("Resolve", func() {
			handler := suite.InferenceRoot()
			var appSettings AppSettings
			err := miruken.Resolve(handler, &appSettings)
			suite.Nil(err)
			suite.NotNil(appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler := suite.InferenceRoot()
			var appSettings []AppSettings
			err := miruken.ResolveAll(handler, &appSettings)
			suite.Nil(err)
			suite.Len(appSettings, 2)
		})
	})

	suite.Run("Named", func () {
		suite.Run("Resolve", func() {
			handler := suite.InferenceRoot()
			var appSettings AppSettings
			err := miruken.Resolve(handler, &appSettings,
				func(c *miruken.ConstraintBuilder) {
					c.Named("local")
				})
			suite.Nil(err)
			suite.IsType(&LocalSettings{}, appSettings)

			err = miruken.Resolve(handler, &appSettings,
				func(c *miruken.ConstraintBuilder) {
					c.Named("remote")
				})
			suite.Nil(err)
			suite.IsType(&RemoteSettings{}, appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler := suite.InferenceRoot()
			var appSettings []AppSettings
			err := miruken.ResolveAll(handler, &appSettings,
				func(c *miruken.ConstraintBuilder) {
					c.Named("remote")
				})
			suite.Nil(err)
			suite.Len(appSettings, 1)
		})

		suite.Run("Inject", func() {
			handler := suite.InferenceRoot()
			var client *Client
			err := miruken.Resolve(handler, &client)
			suite.Nil(err)
			suite.NotNil(client)
			suite.IsType(&LocalSettings{}, client.Local())
			suite.IsType(&RemoteSettings{}, client.Remote())
		})
	})

	suite.Run("Metadata", func () {
		suite.Run("Resolve", func() {
			handler := suite.InferenceRoot()
			var doctor Person
			err := miruken.Resolve(handler, &doctor,
				func(c *miruken.ConstraintBuilder) {
					c.WithConstraint(new(Doctor))
				})
			suite.Nil(err)
			suite.NotNil(doctor)
			suite.Equal("Jack", doctor.FirstName())
			suite.Equal("Zigler", doctor.LastName())

			var programmer Person
			err = miruken.Resolve(handler, &programmer,
				func(c *miruken.ConstraintBuilder) {
					c.WithConstraint(new(Programmer))
				})
			suite.Nil(err)
			suite.NotNil(programmer)
			suite.Equal("Paul", programmer.FirstName())
			suite.Equal("Allen", programmer.LastName())
		})

		suite.Run("ResolveAll", func() {
			handler := suite.InferenceRootWith(reflect.TypeOf((*PersonProvider)(nil)))
			var programmers []Person
			err := miruken.ResolveAll(handler, &programmers,
				func(c *miruken.ConstraintBuilder) {
					c.WithConstraint(new(Programmer))
				})
			suite.Nil(err)
			suite.Len(programmers, 1)
		})
	})
}

func TestConstraintTestSuite(t *testing.T) {
	suite.Run(t, new(ConstraintTestSuite))
}
