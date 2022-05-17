package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"testing"
)

type Person interface {
	FirstName() string
	LastName()  string
}

type PersonData struct {
	firstName string
	lastName  string
}

func (p *PersonData) FirstName() string {
	return p.firstName
}

func (p *PersonData) LastName() string {
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

type Programmer struct {
	miruken.Qualifier[Programmer]
}

type Hospital struct {
	doctor     Person
	programmer Person
}

func (h *Hospital) Constructor(
	_*struct{ Doctor },     doctor     Person,
	_*struct{ Programmer }, programmer Person,
) {
	h.doctor     = doctor
	h.programmer = programmer
}

func (h *Hospital) Doctor() Person {
	return h.doctor
}

func (h *Hospital) Programmer() Person {
	return h.programmer
}

type PersonProvider struct{}

func (p *PersonProvider) Doctor(
	_*struct{
		miruken.Provides
		miruken.Singleton
		Doctor
      },
) Person {
	return &PersonData{"Jack", "Zigler"}
}

func (p *PersonProvider) Programmer(
	_*struct{
		miruken.Provides
		miruken.Singleton
		Programmer
      },
) Person {
	return &PersonData{"Paul", "Allen"}
}

type AppSettings interface {
	ServerUrl() string
}

type LocalSettings struct{}

func (l *LocalSettings) Constructor(
	_*struct{
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
	_*struct{
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
	_*struct{ miruken.Named `name:"local"` },  local  AppSettings,
	_*struct{ miruken.Named `name:"remote"` }, remote AppSettings,
) {
	c.local  = local
	c.remote = remote
}

func (c *Client) Local() AppSettings {
	return c.local
}

func (c *Client) Remote() AppSettings {
	return c.remote
}

type ConstraintTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *ConstraintTestSuite) SetupTest() {
	suite.specs = []any{
		&PersonProvider{},
		&LocalSettings{},
		&RemoteSettings{},
		&Hospital{},
		&Client{},
	}
}

func (suite *ConstraintTestSuite) Setup() miruken.Handler {
	return suite.SetupWith(suite.specs...)
}

func (suite *ConstraintTestSuite) SetupWith(specs ... any) miruken.Handler {
	return miruken.Setup(miruken.WithHandlerSpecs(specs...))
}

func (suite *ConstraintTestSuite) TestConstraints() {
	suite.Run("No Constraints", func () {
		suite.Run("Resolve", func() {
			handler := suite.Setup()
			appSettings, err := miruken.Resolve[AppSettings](handler)
			suite.Nil(err)
			suite.NotNil(appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler := suite.Setup()
			appSettings, err := miruken.ResolveAll[AppSettings](handler)
			suite.Nil(err)
			suite.Len(appSettings, 2)
		})
	})

	suite.Run("Named", func () {
		suite.Run("Resolve", func() {
			handler := suite.Setup()
			appSettings, err := miruken.Resolve[AppSettings](
				handler, miruken.WithName("local"))
			suite.Nil(err)
			suite.IsType(&LocalSettings{}, appSettings)

			appSettings, err = miruken.Resolve[AppSettings](
				handler,  miruken.WithName("remote"))
			suite.Nil(err)
			suite.IsType(&RemoteSettings{}, appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler := suite.Setup()
			appSettings, err := miruken.ResolveAll[AppSettings](
				handler, miruken.WithName("remote"))
			suite.Nil(err)
			suite.Len(appSettings, 1)
		})

		suite.Run("Inject", func() {
			handler := suite.Setup()
			client, err := miruken.Resolve[*Client](handler)
			suite.Nil(err)
			suite.NotNil(client)
			suite.IsType(&LocalSettings{}, client.Local())
			suite.IsType(&RemoteSettings{}, client.Remote())
		})
	})

	suite.Run("Metadata", func () {
		suite.Run("Resolve", func() {
			handler := suite.Setup()
			doctor, err := miruken.Resolve[Person](
				handler, miruken.WithConstraintProvider(&Doctor{}))
			suite.Nil(err)
			suite.NotNil(doctor)
			suite.Equal("Jack", doctor.FirstName())
			suite.Equal("Zigler", doctor.LastName())

			programmer, err := miruken.Resolve[Person](
				handler, miruken.WithQualifier[Programmer]())
			suite.Nil(err)
			suite.NotNil(programmer)
			suite.Equal("Paul", programmer.FirstName())
			suite.Equal("Allen", programmer.LastName())
		})

		suite.Run("ResolveAll", func() {
			handler := suite.Setup()
			programmers, err := miruken.ResolveAll[Person](
				handler,
				func(c *miruken.ConstraintBuilder) {
					c.WithConstraint(new(Programmer))
				})
			suite.Nil(err)
			suite.Len(programmers, 1)
		})

		suite.Run("Inject", func() {
			handler := suite.Setup()
			hospital, err := miruken.Resolve[*Hospital](handler)
			suite.Nil(err)
			suite.NotNil(hospital)
			suite.Equal("Jack", hospital.Doctor().FirstName())
			suite.Equal("Zigler", hospital.Doctor().LastName())
			suite.Equal("Paul", hospital.Programmer().FirstName())
			suite.Equal("Allen", hospital.Programmer().LastName())
		})
	})
}

func TestConstraintTestSuite(t *testing.T) {
	suite.Run(t, new(ConstraintTestSuite))
}
