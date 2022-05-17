package test

import (
	"bytes"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"io"
	"strings"
	"testing"
)

type PlayerMapper struct{}

func (m *PlayerMapper) ToPlayerJson(
	_*struct{
	    miruken.Maps
	    miruken.Format `as:"application/json"`
      }, data PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, data.Name)
}

type JsonTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *JsonTestSuite) SetupTest() {
	suite.specs = []any{
		&miruken.JsonMapper{},
	}
}

func (suite *JsonTestSuite) Setup() miruken.Handler {
	return suite.SetupWith(suite.specs...)
}

func (suite *JsonTestSuite) SetupWith(specs ... any) miruken.Handler {
	return miruken.Setup(miruken.WithHandlerSpecs(specs...))
}

func (suite *JsonTestSuite) TestJson() {
	suite.Run("Maps", func () {
		suite.Run("Json", func() {
			handler := suite.Setup()

			suite.Run("ToJson", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				json, err := miruken.Map[string](handler, data, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", json)
			})

			suite.Run("ToJsonWithOptions", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				json, err := miruken.Map[string](
					miruken.Build(handler, miruken.WithOptions(miruken.JsonOptions{Indent: "  "})),
					data, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", json)
			})

			suite.Run("ToJsonMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				json, err := miruken.Map[string](handler, data, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", json)
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
				err := miruken.MapInto(handler, data, &stream, "application/json")
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
				err := miruken.MapInto(
					miruken.Build(handler, miruken.WithOptions(miruken.JsonOptions{Indent: "  "})),
					data, &stream, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"James Webb\",\n  \"Age\": 85\n}\n", b.String())
			})

			suite.Run("ToJsonOverride", func() {
				handler := suite.SetupWith(
					&miruken.JsonMapper{},
					&PlayerMapper{})

				data :=  PlayerData{
					Id:   1,
					Name: "Tim Howard",
				}

				json, err := miruken.Map[string](handler, data, "application/json")
				suite.Nil(err)
				suite.Equal("{\"id\":1,\"name\":\"Tim Howard\"}", json)
			})

			suite.Run("FromJson", func() {
				type Data struct {
					Name string
					Age  int
				}
				json := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, err := miruken.Map[Data](handler, json, "application/json")
				suite.Nil(err)
				suite.Equal("Ralph Hall", data.Name)
				suite.Equal(84, data.Age)
			})

			suite.Run("FromJsonMap", func() {
				json := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, err := miruken.Map[map[string]any](handler, json, "application/json")
				suite.Nil(err)
				suite.Equal(84.0, data["Age"])
				suite.Equal("Ralph Hall", data["Name"])
			})

			suite.Run("FromJsonStream", func() {
				type Data struct {
					Name string
					Age  int
				}
				stream := strings.NewReader("{\"Name\":\"Ralph Hall\",\"Age\":84}")
				data, err := miruken.Map[Data](handler, stream, "application/json")
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
