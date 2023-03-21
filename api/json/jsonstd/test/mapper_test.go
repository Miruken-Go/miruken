package test

import (
	"bytes"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json/jsonstd"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
	"github.com/stretchr/testify/suite"
	"io"
	"reflect"
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
		maps.It
	    maps.Format `to:"application/json"`
      }, data PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":%q}", data.Id, strings.ToUpper(data.Name))
}

func (m *TypeIdMapper) PlayerDotNet(
	_*struct{
		maps.It
		maps.Format `to:"type:info:dotnet"`
	  }, _ PlayerData,
) api.TypeFieldInfo {
	return api.TypeFieldInfo{TypeField: "$type", TypeValue: "Player,TeamApi"}
}

func (m *TypeIdMapper) TeamDotNet(
	_*struct{
		maps.It
		maps.Format `to:"type:info:dotnet"`
	  }, _ TeamData,
) api.TypeFieldInfo {
	return api.TypeFieldInfo{TypeField: "$type", TypeValue: "Team,TeamApi"}
}

func (m *TypeIdMapper) CreateTeam(
	_*struct{
		creates.It `key:"test.TeamData"`
	  },
) *TeamData {
	return new(TeamData)
}

type JsonStdTestSuite struct {
	suite.Suite
}

func (suite *JsonStdTestSuite) Setup() miruken.Handler {
	handler, _ := miruken.Setup(
		TestFeature,
		jsonstd.Feature()).
		Specs(&api.GoPolymorphism{}).
		Handler()
	return handler
}

func (suite *JsonStdTestSuite) TestJson() {
	suite.Run("It", func () {
		handler := suite.Setup()

		suite.Run("typeInfo", func() {
			suite.Run("TypeId", func() {
				info, _, err := maps.Out[api.TypeFieldInfo](
					handler, PlayerData{}, maps.To("type:info:dotnet", nil))
				suite.Nil(err)
				suite.Equal("$type", info.TypeField)
				suite.Equal("Player,TeamApi", info.TypeValue)
			})
		})

		suite.Run("Json", func() {
			suite.Run("ToJsonBytes", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				byt, _, err := maps.Out[[]byte](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", string(byt))
			})

			suite.Run("ToJsonString", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				j, _, err := maps.Out[string](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", j)
			})

			suite.Run("ToJsonStringPrimitive", func() {
				j, _, err := maps.Out[string](handler, 12, api.ToJson)
				suite.Nil(err)
				suite.Equal("12", j)

				j, _, err = maps.Out[string](handler, "hello", api.ToJson)
				suite.Nil(err)
				suite.Equal("\"hello\"", j)
			})

			suite.Run("ToJsonStringArray", func() {
				j, _, err := maps.Out[string](handler, []int{1,2,3}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[1,2,3]", j)

				j, _, err = maps.Out[string](handler, []string{"A","B","C"}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[\"A\",\"B\",\"C\"]", j)

				j, _, err = maps.Out[string](handler, []any{nil}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[null]", j)
			})

			suite.Run("ToJsonStringWithIndent", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, miruken.Options(jsonstd.Options{Indent: "  "})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", j)
			})

			suite.Run("ToJsonStringMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				j, _, err := maps.Out[string](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", j)
			})

			suite.Run("ToJsonStringTransformers", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, jsonstd.CamelCase),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"id\":9,\"name\":\"Breakaway\",\"players\":[{\"id\":1,\"name\":\"Sean Rose\"},{\"id\":4,\"name\":\"Mark Kingston\"},{\"id\":8,\"name\":\"Michael Binder\"}]}", j)
			})

			suite.Run("ToJsonStringTyped", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}", j)
			})

			suite.Run("ToJsonStringTypedPrimitive", func() {
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					22, api.ToJson)
				suite.Nil(err)
				suite.Equal("22", j)

				j, _, err = maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					"World", api.ToJson)
				suite.Nil(err)
				suite.Equal("\"World\"", j)
			})

			suite.Run("ToJsonStringTypedArray", func() {
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]int{2,4,6}, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"[]int\",\"@values\":[2,4,6]}", j)

				j, _, err = maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]string{"X","Y","Z"}, api.ToJson)
				suite.Equal("{\"@type\":\"[]string\",\"@values\":[\"X\",\"Y\",\"Z\"]}", j)

				j, _, err = maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]any{nil}, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"[]interface {}\",\"@values\":[null]}", j)

				j, _, err = maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
					}}, api.ToJson)
				suite.Equal("{\"@type\":\"[]test.TeamData\",\"@values\":[{\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}", j)

				j, _, err = maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]any{TeamData{Id: 9, Name: "Breakaway", Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
					}}, api.ToJson)
				suite.Equal("{\"@type\":\"[]interface {}\",\"@values\":[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}", j)

				x := []int{1,2}
				y := []TeamData{{
					Id:      1,
					Name:    "Craig",
					Players: nil,
				}}
				fmt.Printf("%T - %v\n", x, reflect.TypeOf(x).String())
				fmt.Printf("%T - %v\n", y, reflect.TypeOf(y).String())
			})

			suite.Run("ToJsonStringTypedIndent", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler,
						api.Polymorphic,
						miruken.Options(jsonstd.Options{Prefix: "abc", Indent:"def"})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\nabcdef\"@type\": \"test.TeamData\",\nabcdef\"Id\": 9,\nabcdef\"Name\": \"Breakaway\",\nabcdef\"Players\": [\nabcdefdef{\nabcdefdefdef\"Id\": 1,\nabcdefdefdef\"Name\": \"Sean Rose\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 4,\nabcdefdefdef\"Name\": \"Mark Kingston\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 8,\nabcdefdefdef\"Name\": \"Michael Binder\"\nabcdefdef}\nabcdef]\nabc}", j)
			})

			suite.Run("ToJsonStringTypedOverrideTypeId", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler,  miruken.Options(api.Options{
						Polymorphism:   miruken.Set(api.PolymorphismRoot),
						TypeInfoFormat: "type:info:dotnet",
					})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"$type\":\"Team,TeamApi\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}", j)
			})

			suite.Run("ToJsonStringTypedTransformers", func() {
				data := TeamData{
					Id: 14,
					Name: "Liverpool",
				}
				j, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic, jsonstd.CamelCase),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"id\":14,\"name\":\"Liverpool\",\"players\":null}", j)
			})

			suite.Run("ToJsonStringOverride", func() {
				data := PlayerData{
					Id:   1,
					Name: "Tim Howard",
				}

				j, _, err := maps.Out[string](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"id\":1,\"name\":\"TIM HOWARD\"}", j)
			})

			suite.Run("ToJsonWriter", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				writer, _, err := maps.Out[io.Writer](handler, data, api.ToJson)
				b := writer.(*bytes.Buffer)
				suite.Nil(err)
				suite.Equal("{\"Name\":\"James Webb\",\"Age\":85}\n", b.String())
			})

			suite.Run("ToJsonIntoWriter", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				var b bytes.Buffer
				writer := io.Writer(&b)
				_, err := maps.Into(handler, data, &writer, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Name\":\"James Webb\",\"Age\":85}\n", b.String())
			})

			suite.Run("ToJsonIntoWriterIndent", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				var b bytes.Buffer
				writer := io.Writer(&b)
				_, err := maps.Into(
					miruken.BuildUp(handler, miruken.Options(jsonstd.Options{Indent: "  "})),
					data, &writer, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"James Webb\",\n  \"Age\": 85\n}\n", b.String())
			})

			suite.Run("ToJsonIntoWriterTransformers", func() {
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
				writer := io.Writer(&b)
				_, err := maps.Into(
					miruken.BuildUp(handler, jsonstd.CamelCase),
					data, &writer, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"id\":15,\"name\":\"Breakaway\",\"players\":[{\"id\":1,\"name\":\"Sean Rose\"},{\"id\":4,\"name\":\"Mark Kingston\"},{\"id\":8,\"name\":\"Michael Binder\"}]}\n", b.String())
			})

			suite.Run("ToJsonIntoWriterJsonTyped", func() {
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
				writer := io.Writer(&b)
				_, err := maps.Into(
					miruken.BuildUp(handler, api.Polymorphic),
					data, &writer, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":15,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}\n", b.String())
			})
			
			suite.Run("FromJsonString", func() {
				type Data struct {
					Name string
					Age  int
				}
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _, err := maps.Out[Data](handler, j, api.FromJson)
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonStringPrimitive", func() {
				i, _, err := maps.Out[int](handler, "9", api.FromJson)
				suite.Nil(err)
				suite.Equal(9, i)

				s, _, err := maps.Out[string](handler, "\"hello\"", api.FromJson)
				suite.Nil(err)
				suite.Equal("hello", s)
			})

			suite.Run("FromJsonStringArray", func() {
				ia, _, err := maps.Out[[]int](handler, "[3,6,9]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]int{3,6,9}, ia)

				sa, _, err := maps.Out[[]string](handler, "[\"E\",\"F\",\"G\"]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"E","F","G"}, sa)
			})

			suite.Run("FromJsonStringMap", func() {
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _,  err := maps.Out[map[string]any](handler, j, api.FromJson)
				suite.Nil(err)
				suite.Equal(84.0, data["Age"])
				suite.Equal("Ralph Hall", data["Name"])
			})

			suite.Run("FromJsonReader", func() {
				type Data struct {
					Name string
					Age  int
				}
				reader := strings.NewReader("{\"Name\":\"Ralph Hall\",\"Age\":84}")
				data, _, err := maps.Out[Data](handler, reader, api.FromJson)
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonByteBuffer", func() {
				type Data struct {
					Name string
					Age  int
				}
				b := bytes.NewBuffer(make([]byte, 0))
				b.Write([]byte("{\"Name\":\"Ralph Hall\",\"Age\":84}"))
				data, _, err := maps.Out[Data](handler, b, api.FromJson)
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonStringTyped", func() {
				j := "{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Liverpool\"}"
				data, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					j, api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Liverpool"}, *data)
			})

			suite.Run("FromJsonStringTypedPrimitive", func() {
				i, _, err := maps.Out[int](
					miruken.BuildUp(handler, api.Polymorphic),
					"99", api.FromJson)
				suite.Nil(err)
				suite.Equal(99, i)

				s, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					"\"world\"", api.FromJson)
				suite.Nil(err)
				suite.Equal("world", s)
			})

			suite.Run("FromJsonStringTypedArray", func() {
				ia, _, err := maps.Out[[]int](
					miruken.BuildUp(handler, api.Polymorphic),
					"[100,200,300]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]int{100,200,300}, ia)

				sa, _, err := maps.Out[[]string](handler, "[\"E\",\"F\",\"G\"]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"E","F","G"}, sa)

				i8a, _, err := maps.Out[[]int8](
					miruken.BuildUp(handler, api.Polymorphic),
					"{\"@type\":\"[]int8\",\"@values\":[2,4,6]}", api.FromJson)
				suite.Nil(err)
				suite.Equal([]int8{2,4,6}, i8a)

				sa, _, err = maps.Out[[]string](
					miruken.BuildUp(handler, api.Polymorphic),
					"{\"@type\":\"[]string\",\"@values\":[\"Craig\",\"Brenda\",\"Lauren\"]}", api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"Craig","Brenda","Lauren"}, sa)

				ta, _, err := maps.Out[[]*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					"{\"@type\":\"[]test.TeamData\",\"@values\":[{\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}", api.FromJson)
				suite.Nil(err)
				suite.Equal([]*TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Sean Rose"},
					{4, "Mark Kingston"},
					{8, "Michael Binder"},
				}}}, ta)

				ta, _, err = maps.Out[[]*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					"[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Luca Schalk\"},{\"Id\":4,\"Name\":\"Brad Bullock\"},{\"Id\":8,\"Name\":\"William Tippet\"}]}]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]*TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Luca Schalk"},
					{4, "Brad Bullock"},
					{8, "William Tippet"},
				}}}, ta)

				tp, _, err := maps.Out[[]any](
					miruken.BuildUp(handler, api.Polymorphic),
					"[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Luca Schalk\"},{\"Id\":4,\"Name\":\"Brad Bullock\"},{\"Id\":8,\"Name\":\"William Tippet\"}]}]", api.FromJson)
				suite.Nil(err)
				suite.Equal([]any{&TeamData{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Luca Schalk"},
					{4, "Brad Bullock"},
					{8, "William Tippet"},
				}}}, tp)
			})

			suite.Run("FromJsonReaderTyped", func() {
				reader := strings.NewReader("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Manchester United\"}")
				data, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					reader, api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Manchester United"}, *data)
			})

			suite.Run("FromJsonReaderTypedLate", func() {
				reader := strings.NewReader("{\"@type\":\"test.TeamData\",\"Id\":11,\"Name\":\"Chelsea\"}")
				late, _, err := maps.Out[miruken.Late](
					miruken.BuildUp(handler, api.Polymorphic),
					reader, api.FromJson)
				suite.Nil(err)
				suite.NotNil(late)
				suite.IsType(&TeamData{}, late.Value)
				suite.Equal(TeamData{Id: 11, Name: "Chelsea"}, *late.Value.(*TeamData))
			})

			suite.Run("FromJsonStringNoTypeInfo", func() {
				j := "{\"Id\":23,\"Name\":\"Everton\"}"
				data, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					j, api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 23, Name: "Everton"}, *data)
			})

			suite.Run("FromJsonStringNoTypeInfoAny", func() {
				j := "{\"Id\":19,\"Name\":\"Wolves\"}"
				dat, _, err := maps.Out[any](
					miruken.BuildUp(handler, api.Polymorphic),
					j, api.FromJson)
				suite.Nil(err)
				suite.True(reflect.DeepEqual(map[string]any{
					"Id": float64(19), "Name": "Wolves",
				}, dat))
			})

			suite.Run("FromJsonStringNoTypeInfoLate", func() {
				j := "{\"Id\":23,\"Name\":\"Everton\"}"
				late, _, err := maps.Out[miruken.Late](
					miruken.BuildUp(handler, api.Polymorphic),
					j, api.FromJson)
				suite.Nil(err)
				suite.True(reflect.DeepEqual(map[string]any{
					"Id": float64(23), "Name": "Everton",
				}, late.Value))
			})

			suite.Run("FromJsonStringMissingTypeInfo", func() {
				j := "{\"@type\":\"test.Team\",\"Id\":9,\"Name\":\"Leeds United\"}"
				_, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					j, api.FromJson)
				suite.NotNil(err)
			})
		})
	})
}

func TestJsonStdTestSuite(t *testing.T) {
	suite.Run(t, new(JsonStdTestSuite))
}
