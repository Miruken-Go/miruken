package test

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/suite"
	"io"
	"miruken.com/miruken"
	"reflect"
	"strings"
	"testing"
)

type JsonTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *JsonTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		reflect.TypeOf((*miruken.JsonMapper)(nil)),
	}
	suite.HandleTypes = handleTypes
}

func (suite *JsonTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *JsonTestSuite) InferenceRootWith(
	handlerTypes ... reflect.Type) miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(handlerTypes...))
}

func (suite *JsonTestSuite) TestJson() {
	suite.Run("Maps", func () {
		suite.Run("Json", func() {
			handler := miruken.NewRootHandler(
				miruken.WithHandlerTypes(reflect.TypeOf((*miruken.JsonMapper)(nil))))

			suite.Run("ToJson", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				var jsonString string
				err := miruken.Map(handler, data, &jsonString, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", jsonString)
			})

			suite.Run("ToJsonWithOptions", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				var jsonString string
				err := miruken.Map(
					miruken.Build(handler, miruken.WithOptions(miruken.JsonOptions{Indent: "  "})),
					data, &jsonString, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", jsonString)
			})

			suite.Run("ToJsonStream", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				var b bytes.Buffer
				stream := io.Writer(&b)
				err := miruken.Map(handler, data, &stream, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Name\":\"James Webb\",\"Age\":85}\n", b.String())
			})

			suite.Run("ToJsonStreamWithOptions", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				var b bytes.Buffer
				stream := io.Writer(&b)
				err := miruken.Map(
					miruken.Build(handler, miruken.WithOptions(miruken.JsonOptions{Indent: "  "})),
					data, &stream, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"James Webb\",\n  \"Age\": 85\n}\n", b.String())
			})

			suite.Run("FromJson", func() {
				type Data struct {
					Name string
					Age  int
				}
				jsonString := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				var data Data
				err := miruken.Map(handler, jsonString, &data, "application/json")
				suite.Nil(err)
				suite.Equal("Ralph Hall", data.Name)
				suite.Equal(84, data.Age)
			})

			suite.Run("FromJsonStream", func() {
				type Data struct {
					Name string
					Age  int
				}
				var data Data
				stream := strings.NewReader("{\"Name\":\"Ralph Hall\",\"Age\":84}")
				err := miruken.Map(handler, stream, &data, "application/json")
				suite.Nil(err)
				suite.Equal("Ralph Hall", data.Name)
				suite.Equal(84, data.Age)
			})
		})

		suite.Run("Invalid", func () {
			failures := 0
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*miruken.HandlerDescriptorError); ok {
						var errMethod miruken.MethodBindingError
						for reason := errors.Unwrap(err.Reason);
							errors.As(reason, &errMethod); reason = errors.Unwrap(reason) {
							failures++
						}
						suite.Equal(6, failures)
					} else {
						suite.Fail("Expected HandlerDescriptorError")
					}
				}
			}()
			miruken.NewRootHandler(
				miruken.WithHandlerTypes(reflect.TypeOf((*InvalidMapper)(nil))))
			suite.Fail("should cause panic")
		})
	})
}

func TestJsonTestSuite(t *testing.T) {
	suite.Run(t, new(JsonTestSuite))
}
