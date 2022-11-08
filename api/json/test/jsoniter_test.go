package test

import (
	"bytes"
	"github.com/json-iterator/go"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/json"
	"github.com/stretchr/testify/suite"
	"io"
	"strings"
	"testing"
)

type JsonIterTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *JsonIterTestSuite) Setup(specs ... any) (miruken.Handler, error) {
	return miruken.Setup(
		json.Feature(json.UseJsonIterator()),
		miruken.HandlerSpecs(specs...))
}

func (suite *JsonIterTestSuite) TestJson() {
	suite.Run("Maps", func () {
		suite.Run("Json", func() {
			handler, _ := suite.Setup()

			suite.Run("ToJson", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				json, _, err := miruken.Map[string](handler, data, "application/json")
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
				j, _, err := miruken.Map[string](
					miruken.BuildUp(handler, miruken.Options(json.IterOptions{
						Config: jsoniter.Config{
							IndentionStep: 2,
						},
					})),
					data, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", j)
			})

			suite.Run("ToJsonMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				j, _, err := miruken.Map[string](
					miruken.BuildUp(handler, miruken.Options(json.IterOptions{
						Config: jsoniter.Config{
							SortMapKeys: true,
						},
					})),
					data, "application/json")
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", j)
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
				_, err := miruken.MapInto(handler, data, &stream, "application/json")
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
				_, err := miruken.MapInto(
					miruken.BuildUp(handler, miruken.Options(json.IterOptions{
						Config: jsoniter.Config{
							IndentionStep: 2,
						},
					})),
					data, &stream, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"James Webb\",\n  \"Age\": 85\n}\n", b.String())
			})

			suite.Run("FromJson", func() {
				type Data struct {
					Name string
					Age  int
				}
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _, err := miruken.Map[Data](handler, j, "application/json")
				suite.Nil(err)
				suite.Equal("Ralph Hall", data.Name)
				suite.Equal(84, data.Age)
			})

			suite.Run("FromJsonMap", func() {
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _,  err := miruken.Map[map[string]any](handler, j, "application/json")
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
				data, _, err := miruken.Map[Data](handler, stream, "application/json")
				suite.Nil(err)
				suite.Equal("Ralph Hall", data.Name)
				suite.Equal(84, data.Age)
			})
		})
	})
}

func TestJsonIterTestSuite(t *testing.T) {
	suite.Run(t, new(JsonIterTestSuite))
}
