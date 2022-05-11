package test

import (
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
) {
	if setup.CanInstall(reflect.TypeOf(i)) {
		i.count++
	}
}

type RegisterTestSuite struct {
	suite.Suite
}

func (suite *RegisterTestSuite) TestSetup() {
	suite.Run("#RegisterHandlers", func () {
		handler := miruken.Setup(
			miruken.WithHandlerSpecs(&MultiHandler{}),
		)

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)

		result = handler.Handle(&Baz{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})

	suite.Run("#Exclude", func () {
		handler := miruken.Setup(
			TestFeature,
			miruken.ExcludeHandlerSpecs(
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
				}),
		)

		var m *MultiHandler
		err := miruken.Resolve(handler, &m)
		suite.Nil(err)
		suite.Nil(m)

		var e *EverythingHandler
		err = miruken.Resolve(handler, &e)
		suite.Nil(err)
		suite.Nil(e)
	})

	suite.Run("#DisableInference", func () {
		handler := miruken.Setup(
			miruken.WithHandlerSpecs(&MultiHandler{}),
			miruken.DisableInference,
		)

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)

		var m *MultiHandler
		err := miruken.Resolve(handler, &m)
		suite.Nil(err)
		suite.Nil(m)
	})

	suite.Run("Installs Once", func () {
		installer := &MyInstaller{}
		miruken.Setup(installer, installer)
		suite.Equal(1, installer.count)
	})
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}