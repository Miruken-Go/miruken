package test

import (
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"strings"
	"testing"
)

type MyInstaller struct {
	count int
}

func (i *MyInstaller) Install(
	setup *miruken.SetupBuilder,
) error {
	if setup.CanInstall(reflect.TypeOf(i)) {
		i.count++
		setup.Specs(&MultiHandler{})
	}
	return nil
}

type RootInstaller struct {}

func (i *RootInstaller) DependsOn() []miruken.Feature {
	return []miruken.Feature{&MyInstaller{}}
}

func (i *RootInstaller) Install(
	setup *miruken.SetupBuilder,
) error {
	return nil
}

type BadInstaller struct {}

func (i BadInstaller) Install(
	*miruken.SetupBuilder,
) error {
	return errors.New("insufficient resources")
}

func (i BadInstaller) AfterInstall(
	*miruken.SetupBuilder, miruken.Handler,
) error {
	return errors.New("process failed to start")
}

type SetupTestSuite struct {
	suite.Suite
}

func (suite *SetupTestSuite) TestSetup() {
	suite.Run("Specs", func () {
		handler, _ := miruken.Setup().Specs(&MultiHandler{}).Handler();

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)

		result = handler.Handle(&Baz{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})

	suite.Run("ExcludeSpecs", func () {
		handler, _ := miruken.Setup(TestFeature).ExcludeSpecs(
				func(spec miruken.HandlerSpec) bool {
					switch ts := spec.(type) {
					case miruken.HandlerTypeSpec:
						name := ts.Name()
						return name == "MultiHandler" || strings.Contains(name, "Invalid")
					default:
						return false
					}
				},
				func(spec miruken.HandlerSpec) bool {
					if ts, ok := spec.(miruken.HandlerTypeSpec); ok {
						return ts.Type() == miruken.TypeOf[*EverythingHandler]()
					}
					return false
				}).Handler()

		m, _, err := miruken.Resolve[*MultiHandler](handler)
		suite.Nil(err)
		suite.Nil(m)

		e, _, err := miruken.Resolve[*EverythingHandler](handler)
		suite.Nil(err)
		suite.Nil(e)
	})

	suite.Run("NoInference", func () {
		handler, _ := miruken.Setup().
			Specs(&MultiHandler{}).
			NoInference().
			Handler()

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)

		m, _, err := miruken.Resolve[*MultiHandler](handler)
		suite.Nil(err)
		suite.Nil(m)
	})

	suite.Run("Installs Once", func () {
		installer := &MyInstaller{}
		handler, err := miruken.Setup(installer, installer).Handler()
		suite.Nil(err)
		suite.Equal(1, installer.count)
		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
	})

	suite.Run("Installs Dependencies", func () {
		handler, err := miruken.Setup(&RootInstaller{}).Handler()
		suite.Nil(err)
		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		multi, _, err := miruken.Resolve[*MultiHandler](handler)
		suite.Nil(err)
		suite.NotNil(multi)
	})

	suite.Run("Overrides Dependencies", func () {
		installer := &MyInstaller{10}
		handler, err := miruken.Setup(&RootInstaller{}, installer).Handler()
		suite.Nil(err)
		suite.NotNil(handler)
		suite.Equal(11, installer.count)
	})

	suite.Run("Errors", func () {
		installer := BadInstaller{}
		_, err := miruken.Setup(installer).Handler()
		suite.Equal("2 errors occurred:\n\t* insufficient resources\n\t* process failed to start\n\n", err.Error())
	})
}

func TestSetupTestSuite(t *testing.T) {
	suite.Run(t, new(SetupTestSuite))
}