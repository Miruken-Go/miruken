package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
)

type MyInstaller struct {
	count int
}

func (i *MyInstaller) Install(
	registration *miruken.RegistrationBuilder,
) {
	if registration.CanInstall(reflect.TypeOf(i)) {
		i.count++
	}
}

type RegisterTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *RegisterTestSuite) TestRegistration() {
	suite.Run("#AddHandlerTypes", func () {
		handler := miruken.NewRegistration(
			miruken.WithHandlerTypes(miruken.TypeOf[*MultiHandler]()),
		).Build()

		result := handler.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)

		result = handler.Handle(&Baz{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})

	suite.Run("#Exclude", func () {
		handler := miruken.NewRegistration(
			miruken.WithHandlerTypes(suite.HandleTypes...),
			miruken.ExcludeHandlerTypes(
				func(t reflect.Type) bool {
					return t.Kind() == reflect.Ptr && t.Elem().Name() == "MultiHandler"
				},
				func(t reflect.Type) bool {
					return t == miruken.TypeOf[*EverythingHandler]()
				}),
		).Build()

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
		handler := miruken.NewRegistration(
			miruken.WithHandlerTypes(suite.HandleTypes...),
			miruken.DisableInference,
		).Build()

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
		miruken.NewRegistration(installer, installer)
		suite.Equal(1, installer.count)
	})
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}