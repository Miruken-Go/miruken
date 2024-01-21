package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraint"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	Person interface {
		FirstName() string
		LastName() string
	}

	PersonData struct {
		firstName string
		lastName  string
	}

	Doctor struct {
		miruken.Metadata
	}

	Programmer struct {
		miruken.Qualifier[Programmer]
	}

	Hospital struct {
		doctor     Person
		programmer Person
	}

	PersonProvider struct{}

	NoConstraintProvider struct{}

	AppSettings interface {
		ServerUrl() string
	}

	LocalSettings struct{}

	RemoteSettings struct{}

	Client struct {
		local  AppSettings
		remote AppSettings
	}
)

// PersonData

func (p *PersonData) FirstName() string {
	return p.firstName
}

func (p *PersonData) LastName() string {
	return p.lastName
}

// Doctor

func (d *Doctor) Init() error {
	d.Metadata = map[any]any{"Job": "Doctor"}
	return nil
}

// Hospital

func (h *Hospital) Constructor(
	_ *struct{ Doctor }, doctor Person,
	_ *struct{ Programmer }, programmer Person,
) {
	h.doctor = doctor
	h.programmer = programmer
}

func (h *Hospital) Doctor() Person {
	return h.doctor
}

func (h *Hospital) Programmer() Person {
	return h.programmer
}

// PersonProvider

func (p *PersonProvider) Doctor(
	_ *struct {
		provides.It
		provides.Single
		Doctor
	},
) Person {
	return &PersonData{"Jack", "Zigler"}
}

func (p *PersonProvider) Programmer(
	_ *struct {
		provides.It
		provides.Single
		Programmer
	},
) Person {
	return &PersonData{"Paul", "Allen"}
}

// NoConstraintProvider

func (n *NoConstraintProvider) Person(
	_ *provides.It,
) Person {
	return &PersonData{"Benjamin", "Franklin"}
}

// LocalSettings

func (l *LocalSettings) Constructor(
	_ *struct {
		provides.It
		provides.Single
		constraint.Named `name:"local"`
	},
) {
}

func (l *LocalSettings) ServerUrl() string {
	return "http://localhost/Server"
}

// RemoteSettings

func (r *RemoteSettings) Constructor(
	_ *struct {
		provides.It
		provides.Single
		constraint.Named `name:"remote"`
	},
) {
}

func (r *RemoteSettings) ServerUrl() string {
	return "https://remote/Server"
}

// Client

func (c *Client) Constructor(
	_ *struct {
		constraint.Named `name:"local"`
	}, local AppSettings,
	_ *struct {
		constraint.Named `name:"remote"`
	}, remote AppSettings,
) {
	c.local = local
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

func (suite *ConstraintTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *ConstraintTestSuite) TestConstraints() {
	suite.Run("No Constraints", func() {
		suite.Run("Resolve", func() {
			handler, _ := suite.Setup()
			appSettings, _, ok, err := miruken.Resolve[AppSettings](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler, _ := suite.Setup()
			appSettings, _, err := miruken.ResolveAll[AppSettings](handler)
			suite.Nil(err)
			suite.Len(appSettings, 2)
		})

		suite.Run("Context", func() {
			handler, _ := suite.Setup(&NoConstraintProvider{})
			person, _, ok, err := miruken.Resolve[Person](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(person)
			person, _, ok, err = miruken.Resolve[Person](handler, internal.New[Doctor]())
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(person)
		})
	})

	suite.Run("Named", func() {
		suite.Run("Resolve", func() {
			handler, _ := suite.Setup()
			appSettings, _, ok, err := miruken.Resolve[AppSettings](handler, "local")
			suite.True(ok)
			suite.Nil(err)
			suite.IsType(&LocalSettings{}, appSettings)

			appSettings, _, ok, err = miruken.Resolve[AppSettings](handler, "remote")
			suite.True(ok)
			suite.Nil(err)
			suite.IsType(&RemoteSettings{}, appSettings)
		})

		suite.Run("ResolveAll", func() {
			handler, _ := suite.Setup()
			appSettings, _, err := miruken.ResolveAll[AppSettings](handler, "remote")
			suite.Nil(err)
			suite.Len(appSettings, 1)
		})

		suite.Run("Inject", func() {
			handler, _ := suite.Setup()
			client, _, ok, err := miruken.Resolve[*Client](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(client)
			suite.IsType(&LocalSettings{}, client.Local())
			suite.IsType(&RemoteSettings{}, client.Remote())
		})
	})

	suite.Run("Metadata", func() {
		suite.Run("Resolve", func() {
			handler, _ := suite.Setup()
			doctor, _, ok, err := miruken.Resolve[Person](handler, internal.New[Doctor]())
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(doctor)
			suite.Equal("Jack", doctor.FirstName())
			suite.Equal("Zigler", doctor.LastName())

			programmer, _, ok, err := miruken.Resolve[Person](handler, Programmer{})
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(programmer)
			suite.Equal("Paul", programmer.FirstName())
			suite.Equal("Allen", programmer.LastName())
		})

		suite.Run("ResolveAll", func() {
			handler, _ := suite.Setup()
			programmers, _, err := miruken.ResolveAll[Person](handler, new(Programmer))
			suite.Nil(err)
			suite.Len(programmers, 1)
		})

		suite.Run("Inject", func() {
			handler, _ := suite.Setup()
			hospital, _, ok, err := miruken.Resolve[*Hospital](handler)
			suite.True(ok)
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
