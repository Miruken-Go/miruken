package test

import (
	"bytes"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"io"
	"reflect"
	"strings"
	"testing"
)

type PlayerMapper struct{}

func (m *PlayerMapper) ToPlayerJson(
	_ *struct{
	miruken.Maps
	miruken.Format `as:"application/json"`
}, data PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, data.Name)
}

type JsonTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *JsonTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		miruken.TypeOf[*miruken.JsonMapper](),
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
			handler := suite.InferenceRoot()

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

			suite.Run("ToJsonMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				var jsonString string
				err := miruken.Map(handler, data, &jsonString, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", jsonString)
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

			suite.Run("ToJsonOverride", func() {
				handler := suite.InferenceRootWith(
					miruken.TypeOf[*miruken.JsonMapper](),
					miruken.TypeOf[*PlayerMapper]())

				data :=  PlayerData{
					Id:   1,
					Name: "Tim Howard",
				}
				var jsonString string
				err := miruken.Map(handler, data, &jsonString, "application/json")
				suite.Nil(err)
				suite.Equal("{\"id\":1,\"name\":\"Tim Howard\"}", jsonString)
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

			suite.Run("FromJsonMap", func() {
				jsonString := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				var data map[string]any
				err := miruken.Map(handler, jsonString, &data, "application/json")
				suite.Nil(err)
				suite.Equal(84.0, data["Age"])
				suite.Equal("Ralph Hall", data["Name"])
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
	})
}

func TestJsonTestSuite(t *testing.T) {
	suite.Run(t, new(JsonTestSuite))
}
