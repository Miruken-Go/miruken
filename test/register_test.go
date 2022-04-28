package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"strings"
	"testing"
)

type RegisterTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *RegisterTestSuite) SetupTest() {
	suite.HandleTypes = make([]reflect.Type, 0)
	for _, typ := range HandlerTestTypes {
		if !strings.Contains(typ.Elem().Name(), "Invalid") {
			suite.HandleTypes = append(suite.HandleTypes, typ)
		}
	}
}

func (suite *RegisterTestSuite) TestRegistration() {
	suite.Run("WithHandlerTypes", func () {
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

	suite.Run("ExcludeHandlerTypes", func () {
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

	suite.Run("DisableInference", func () {
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
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}