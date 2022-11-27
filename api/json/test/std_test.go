package test

import (
	"bytes"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json"
	"github.com/stretchr/testify/suite"
	"io"
	"strings"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	PlayerMapper struct{}

	PlayerData struct {
		Id   int32
		Name string
	}

	TeamData struct {
		Id      int32
		Name    string
		Players []PlayerData
	}

	TypeIdMapper struct {}
)

// PlayerMapper

func (m *PlayerMapper) ToPlayerJson(
	_*struct{
	    miruken.Maps
	    miruken.Format `to:"application/json"`
      }, data PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, strings.ToUpper(data.Name))
}

func (m *TypeIdMapper) PlayerDotNet(
	_*struct{
		miruken.Maps
		miruken.Format `to:"type:info"`
	  }, _ PlayerData,
) json.TypeFieldInfo {
	return json.TypeFieldInfo{Field: "$type", Value: "Player,TeamApi"}
}

func (m *TypeIdMapper) CreateTeam(
	_*struct{
		miruken.Creates `key:"test.TeamData"`
	  }, _ *miruken.Creates,
) *TeamData {
	return &TeamData{}
}

type JsonStdTestSuite struct {
	suite.Suite
}

func (suite *JsonStdTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		json.Feature(json.UseStandard()),
		miruken.HandlerSpecs(&json.GoTypeFieldMapper{}))
	return handler
}

func (suite *JsonStdTestSuite) TestJson() {
	suite.Run("Maps", func () {
		handler := suite.Setup()

		suite.Run("TypeInfo", func() {
			suite.Run("TypeId", func() {
				info, _, err := miruken.Map[json.TypeFieldInfo](
					handler, PlayerData{}, miruken.To("type:info"))
				suite.Nil(err)
				suite.Equal("$type", info.Field)
				suite.Equal("Player,TeamApi", info.Value)
			})
		})

		suite.Run("Json", func() {
			suite.Run("ToJson", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				j, _, err := miruken.Map[string](handler, data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", j)
			})

			suite.Run("ToJsonWithIndent", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				j, _, err := miruken.Map[string](
					miruken.BuildUp(handler, miruken.Options(json.StdOptions{Indent: "  "})),
					data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", j)
			})

			suite.Run("ToJsonMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				j, _, err := miruken.Map[string](handler, data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", j)
			})

			suite.Run("ToJsonTyped", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := miruken.Map[string](
					miruken.BuildUp(handler, miruken.Options(
						api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
					data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}", j)
			})

			suite.Run("ToJsonTypedIndent", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := miruken.Map[string](
					miruken.BuildUp(handler,
						miruken.Options(api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)}),
						miruken.Options(json.StdOptions{Prefix: "abc", Indent: "def"})),
					data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\nabcdef\"@type\": \"test.TeamData\",\nabcdef\"Id\": 9,\nabcdef\"Name\": \"Breakaway\",\nabcdef\"Players\": [\nabcdefdef{\nabcdefdefdef\"Id\": 1,\nabcdefdefdef\"Name\": \"Sean Rose\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 4,\nabcdefdefdef\"Name\": \"Mark Kingston\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 8,\nabcdefdefdef\"Name\": \"Michael Binder\"\nabcdefdef}\nabcdef]\nabc}", j)
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
				_, err := miruken.MapInto(handler, data, &stream, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"Name\":\"James Webb\",\"Age\":85}\n", b.String())
			})

			suite.Run("ToJsonStreamIndent", func() {
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
					miruken.BuildUp(handler, miruken.Options(json.StdOptions{Indent: "  "})),
					data, &stream, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"James Webb\",\n  \"Age\": 85\n}\n", b.String())
			})

			suite.Run("ToJsonStreamTyped", func() {
				data := TeamData{
					Id: 15,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				var b bytes.Buffer
				stream := io.Writer(&b)
				_, err := miruken.MapInto(
					miruken.BuildUp(handler,
						miruken.Options(api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
					data, &stream, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":15,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}\n", b.String())
			})

			suite.Run("ToJsonStreamTypedIndent", func() {
				data := TeamData{
					Id: 15,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				var b bytes.Buffer
				stream := io.Writer(&b)
				_, err := miruken.MapInto(
					miruken.BuildUp(handler,
						miruken.Options(api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)}),
						miruken.Options(json.StdOptions{Prefix: "abc", Indent: "def"})),
					data, &stream, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\nabcdef\"@type\": \"test.TeamData\",\nabcdef\"Id\": 15,\nabcdef\"Name\": \"Breakaway\",\nabcdef\"Players\": [\nabcdefdef{\nabcdefdefdef\"Id\": 1,\nabcdefdefdef\"Name\": \"Sean Rose\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 4,\nabcdefdefdef\"Name\": \"Mark Kingston\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 8,\nabcdefdefdef\"Name\": \"Michael Binder\"\nabcdefdef}\nabcdef]\nabc}\n", b.String())
			})

			suite.Run("ToJsonOverride", func() {
				data := PlayerData{
					Id:   1,
					Name: "Tim Howard",
				}

				j, _, err := miruken.Map[string](handler, data, miruken.To("application/json"))
				suite.Nil(err)
				suite.Equal("{\"id\":1,\"name\":\"TIM HOWARD\"}", j)
			})

			suite.Run("ToJsonMissingTypeInfo", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				_, _, err := miruken.Map[string](
					miruken.BuildUp(handler, miruken.Options(
						api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
						data, miruken.To("application/json"))
				suite.NotNil(err)
				suite.ErrorContains(err, "no type info for anonymous struct { Name string; Age int }")
			})

			suite.Run("FromJson", func() {
				type Data struct {
					Name string
					Age  int
				}
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _, err := miruken.Map[Data](handler, j, miruken.From("application/json"))
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonMap", func() {
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _,  err := miruken.Map[map[string]any](handler, j, miruken.From("application/json"))
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
				data, _, err := miruken.Map[Data](handler, stream, miruken.From("application/json"))
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonTyped", func() {
				j := "{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Olivier Giroud\"}"
				data, _, err := miruken.Map[*TeamData](
					miruken.BuildUp(handler, miruken.Options(
						api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
					j, miruken.From("application/json"))
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Olivier Giroud"}, *data)
			})

			suite.Run("FromJsonStreamTyped", func() {
				stream := strings.NewReader("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Olivier Giroud\"}")
				data, _, err := miruken.Map[*TeamData](
					miruken.BuildUp(handler, miruken.Options(
						api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
					stream, miruken.From("application/json"))
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Olivier Giroud"}, *data)
			})

			suite.Run("FromJsonMissingTypeInfo", func() {
				j := "{\"@type\":\"test.Team\",\"Id\":9,\"Name\":\"Olivier Giroud\"}"
				_, _, err := miruken.Map[*TeamData](
					miruken.BuildUp(handler, miruken.Options(
						api.PolymorphicOptions{PolymorphicHandling: miruken.Set(api.PolymorphicHandlingRoot)})),
					j, miruken.From("application/json"))
				suite.NotNil(err)
			})
		})
	})
}

func TestJsonStdTestSuite(t *testing.T) {
	suite.Run(t, new(JsonStdTestSuite))
}
