package test

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"reflect"
	"strings"
	"testing"
)

type MyInstaller struct {
	count int
}

func (i *MyInstaller) Install(
	setup *setup.Builder,
) error {
	if setup.Tag(reflect.TypeOf(i)) {
		i.count++
		setup.Specs(&MultiHandler{})
	}
	return nil
}

type RootInstaller struct {}

func (i *RootInstaller) DependsOn() []setup.Feature {
	return []setup.Feature{&MyInstaller{}}
}

func (i *RootInstaller) Install(
	setup *setup.Builder,
) error {
	return nil
}

type BadInstaller struct {}

func (i BadInstaller) Install(
	*setup.Builder,
) error {
	return errors.New("insufficient resources")
}

func (i BadInstaller) AfterInstall(
	*setup.Builder, miruken.Handler,
) error {
	return errors.New("process failed to start")
}

type SetupTestSuite struct {
	suite.Suite
}

func (suite *SetupTestSuite) TestSetup() {
	suite.Run("Specs", func () {
		handler, _ := setup.New().Specs(&MultiHandler{}).Context()

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)

		result = handler.Handle(&Baz{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})

	suite.Run("ExcludeSpecs", func () {
		handler, _ := setup.New(TestFeature).ExcludeSpecs(
				func(spec miruken.HandlerSpec) bool {
					switch ts := spec.(type) {
					case miruken.TypeSpec:
						name := ts.Name()
						return name == "MultiHandler" || strings.Contains(name, "Invalid")
					default:
						return false
					}
				},
				func(spec miruken.HandlerSpec) bool {
					if ts, ok := spec.(miruken.TypeSpec); ok {
						return ts.Type() == internal.TypeOf[*EverythingHandler]()
					}
					return false
				}).Context()

		m, _, ok, err := miruken.Resolve[*MultiHandler](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(m)

		e, _, ok, err := miruken.Resolve[*EverythingHandler](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(e)
	})

	suite.Run("WithoutInference", func () {
		handler, _ := setup.New().
			WithoutInference().
			Specs(&MultiHandler{}).
			Context()

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)

		m, _, ok, err := miruken.Resolve[*MultiHandler](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(m)
	})

	suite.Run("Installs once", func () {
		installer := &MyInstaller{}
		handler, err := setup.New(installer, installer).Context()
		suite.Nil(err)
		suite.Equal(1, installer.count)
		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
	})

	suite.Run("Installs Dependencies", func () {
		handler, err := setup.New(&RootInstaller{}).Context()
		suite.Nil(err)
		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		multi, _, ok, err := miruken.Resolve[*MultiHandler](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(multi)
	})

	suite.Run("Overrides Dependencies", func () {
		installer := &MyInstaller{10}
		handler, err := setup.New(&RootInstaller{}, installer).Context()
		suite.Nil(err)
		suite.NotNil(handler)
		suite.Equal(11, installer.count)
	})

	suite.Run("Errors", func () {
		installer := BadInstaller{}
		_, err := setup.New(installer).Context()
		suite.Equal("2 errors occurred:\n\t* insufficient resources\n\t* process failed to start\n\n", err.Error())
	})
}

func TestSetupTestSuite(t *testing.T) {
	suite.Run(t, new(SetupTestSuite))
}