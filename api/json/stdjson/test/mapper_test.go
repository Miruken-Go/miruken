package test

import (
	"bytes"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json/stdjson"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/setup"
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
) []byte {
	return []byte(fmt.Sprintf("{\"id\":%v,\"name\":%q}", data.Id, strings.ToUpper(data.Name)))
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

type StdJsonTestSuite struct {
	suite.Suite
}

func (suite *StdJsonTestSuite) Setup() miruken.Handler {
	handler, _ := setup.New(
		TestFeature,
		stdjson.Feature()).
		Specs(&api.GoPolymorphism{}).
		Handler()
	return handler
}

func (suite *StdJsonTestSuite) TestJson() {
	suite.Run("It", func () {
		handler := suite.Setup()

		suite.Run("typeInfo", func() {
			suite.Run("TypeId", func() {
				info, _, _, err := maps.Out[api.TypeFieldInfo](
					handler, PlayerData{}, maps.To("type:info:dotnet", nil))
				suite.Nil(err)
				suite.Equal("$type", info.TypeField)
				suite.Equal("Player,TeamApi", info.TypeValue)
			})
		})

		suite.Run("Json", func() {
			suite.Run("ToJsonBytesStruct", func() {
				data := struct{
					Name string
					Age  int
				}{
					"John Smith",
					23,
				}
				b, _, _, err := maps.Out[[]byte](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Name\":\"John Smith\",\"Age\":23}", string(b))
			})

			suite.Run("ToJsonBytesPrimitive", func() {
				b, _, _, err := maps.Out[[]byte](handler, 12, api.ToJson)
				suite.Nil(err)
				suite.Equal("12", string(b))

				b, _, _, err = maps.Out[[]byte](handler, "hello", api.ToJson)
				suite.Nil(err)
				suite.Equal("\"hello\"", string(b))
			})

			suite.Run("ToJsonBytesArray", func() {
				b, _, _, err := maps.Out[[]byte](handler, []int{1,2,3}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[1,2,3]", string(b))

				b, _, _, err = maps.Out[[]byte](handler, []string{"A","B","C"}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[\"A\",\"B\",\"C\"]", string(b))

				b, _, _, err = maps.Out[[]byte](handler, []any{nil}, api.ToJson)
				suite.Nil(err)
				suite.Equal("[null]", string(b))
			})

			suite.Run("ToJsonBytesWithIndent", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, miruken.Options(stdjson.Options{Indent: "  "})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", string(b))
			})

			suite.Run("ToJsonBytesMap", func() {
				data := map[string]any{
					"Id":    2,
					"Name": "George Best",
				}
				b, _, _, err := maps.Out[[]byte](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"Id\":2,\"Name\":\"George Best\"}", string(b))
			})

			suite.Run("ToJsonBytesTransformers", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, stdjson.CamelCase),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"id\":9,\"name\":\"Breakaway\",\"players\":[{\"id\":1,\"name\":\"Sean Rose\"},{\"id\":4,\"name\":\"Mark Kingston\"},{\"id\":8,\"name\":\"Michael Binder\"}]}", string(b))
			})

			suite.Run("ToJsonBytesTyped", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}", string(b))
			})

			suite.Run("ToJsonBytesTypedPrimitive", func() {
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					22, api.ToJson)
				suite.Nil(err)
				suite.Equal("22", string(b))

				b, _, _, err = maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					"World", api.ToJson)
				suite.Nil(err)
				suite.Equal("\"World\"", string(b))
			})

			suite.Run("ToJsonBytesTypedArray", func() {
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					[]int{2,4,6}, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"[]int\",\"@values\":[2,4,6]}", string(b))

				b, _, _, err = maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					[]string{"X","Y","Z"}, api.ToJson)
				suite.Equal("{\"@type\":\"[]string\",\"@values\":[\"X\",\"Y\",\"Z\"]}", string(b))

				b, _, _, err = maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					[]any{nil}, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"[]interface {}\",\"@values\":[null]}", string(b))

				b, _, _, err = maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					[]TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
					}}, api.ToJson)
				suite.Equal("{\"@type\":\"[]test.TeamData\",\"@values\":[{\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}", string(b))

				b, _, _, err = maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic),
					[]any{TeamData{Id: 9, Name: "Breakaway", Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
					}}, api.ToJson)
				suite.Equal("{\"@type\":\"[]interface {}\",\"@values\":[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}", string(b))

				x := []int{1,2}
				y := []TeamData{{
					Id:      1,
					Name:    "Craig",
					Players: nil,
				}}
				fmt.Printf("%T - %v\n", x, reflect.TypeOf(x).String())
				fmt.Printf("%T - %v\n", y, reflect.TypeOf(y).String())
			})

			suite.Run("ToJsonBytesTypedIndent", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler,
						api.Polymorphic,
						miruken.Options(stdjson.Options{Prefix: "abc", Indent:"def"})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\nabcdef\"@type\": \"test.TeamData\",\nabcdef\"Id\": 9,\nabcdef\"Name\": \"Breakaway\",\nabcdef\"Players\": [\nabcdefdef{\nabcdefdefdef\"Id\": 1,\nabcdefdefdef\"Name\": \"Sean Rose\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 4,\nabcdefdefdef\"Name\": \"Mark Kingston\"\nabcdefdef},\nabcdefdef{\nabcdefdefdef\"Id\": 8,\nabcdefdefdef\"Name\": \"Michael Binder\"\nabcdefdef}\nabcdef]\nabc}", string(b))
			})

			suite.Run("ToJsonBytesTypedOverrideTypeId", func() {
				data := TeamData{
					Id: 9,
					Name: "Breakaway",
					Players: []PlayerData{
						{1, "Sean Rose"},
						{4, "Mark Kingston"},
						{8, "Michael Binder"},
					},
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler,  miruken.Options(api.Options{
						Polymorphism:   miruken.Set(api.PolymorphismRoot),
						TypeInfoFormat: "type:info:dotnet",
					})),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"$type\":\"Team,TeamApi\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}", string(b))
			})

			suite.Run("ToJsonBytesTypedTransformers", func() {
				data := TeamData{
					Id: 14,
					Name: "Liverpool",
				}
				b, _, _, err := maps.Out[[]byte](
					miruken.BuildUp(handler, api.Polymorphic, stdjson.CamelCase),
					data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"id\":14,\"name\":\"Liverpool\",\"players\":null}", string(b))
			})

			suite.Run("ToJsonBytesOverride", func() {
				data := PlayerData{
					Id:   1,
					Name: "Tim Howard",
				}

				b, _, _, err := maps.Out[[]byte](handler, data, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"id\":1,\"name\":\"TIM HOWARD\"}", string(b))
			})

			suite.Run("ToJsonWriter", func() {
				data := struct{
					Name string
					Age  int
				}{
					"James Webb",
					85,
				}
				writer, _, _, err := maps.Out[io.Writer](handler, data, api.ToJson)
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
				_, _, err := maps.Into(handler, data, &writer, api.ToJson)
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
				_, _, err := maps.Into(
					miruken.BuildUp(handler, miruken.Options(stdjson.Options{Indent: "  "})),
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
				_, _, err := maps.Into(
					miruken.BuildUp(handler, stdjson.CamelCase),
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
				_, _, err := maps.Into(
					miruken.BuildUp(handler, api.Polymorphic),
					data, &writer, api.ToJson)
				suite.Nil(err)
				suite.Equal("{\"@type\":\"test.TeamData\",\"Id\":15,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}\n", b.String())
			})
			
			suite.Run("FromJsonBytesStruct", func() {
				type Data struct {
					Name string
					Age  int
				}
				b := []byte("{\"Name\":\"Ralph Hall\",\"Age\":84}")
				data, _, _, err := maps.Out[Data](handler, b, api.FromJson)
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonBytesPrimitive", func() {
				i, _, _, err := maps.Out[int](handler, []byte("9"), api.FromJson)
				suite.Nil(err)
				suite.Equal(9, i)

				s, _, _, err := maps.Out[string](handler, []byte("\"hello\""), api.FromJson)
				suite.Nil(err)
				suite.Equal("hello", s)
			})

			suite.Run("FromJsonBytesArray", func() {
				ia, _, _, err := maps.Out[[]int](handler, []byte("[3,6,9]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]int{3,6,9}, ia)

				sa, _, _, err := maps.Out[[]string](handler, []byte("[\"E\",\"F\",\"G\"]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"E","F","G"}, sa)
			})

			suite.Run("FromJsonBytesMap", func() {
				j := "{\"Name\":\"Ralph Hall\",\"Age\":84}"
				data, _, _, err := maps.Out[map[string]any](handler, []byte(j), api.FromJson)
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
				data, _, _, err := maps.Out[Data](handler, reader, api.FromJson)
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
				data, _, _, err := maps.Out[Data](handler, b, api.FromJson)
				suite.Nil(err)
				suite.Equal(Data{Name: "Ralph Hall", Age: 84}, data)
			})

			suite.Run("FromJsonBytesTyped", func() {
				j := "{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Liverpool\"}"
				data, _, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte(j), api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Liverpool"}, *data)
			})

			suite.Run("FromJsonBytesTypedPrimitive", func() {
				i, _, _, err := maps.Out[int](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("99"), api.FromJson)
				suite.Nil(err)
				suite.Equal(99, i)

				s, _, _, err := maps.Out[string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("\"world\""), api.FromJson)
				suite.Nil(err)
				suite.Equal("world", s)
			})

			suite.Run("FromJsonBytesTypedArray", func() {
				ia, _, _, err := maps.Out[[]int](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("[100,200,300]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]int{100,200,300}, ia)

				sa, _, _, err := maps.Out[[]string](handler, []byte("[\"E\",\"F\",\"G\"]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"E","F","G"}, sa)

				i8a, _, _, err := maps.Out[[]int8](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("{\"@type\":\"[]int8\",\"@values\":[2,4,6]}"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]int8{2,4,6}, i8a)

				sa, _, _, err = maps.Out[[]string](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("{\"@type\":\"[]string\",\"@values\":[\"Craig\",\"Brenda\",\"Lauren\"]}"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]string{"Craig","Brenda","Lauren"}, sa)

				ta, _, _, err := maps.Out[[]*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("{\"@type\":\"[]test.TeamData\",\"@values\":[{\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Sean Rose\"},{\"Id\":4,\"Name\":\"Mark Kingston\"},{\"Id\":8,\"Name\":\"Michael Binder\"}]}]}"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]*TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Sean Rose"},
					{4, "Mark Kingston"},
					{8, "Michael Binder"},
				}}}, ta)

				ta, _, _, err = maps.Out[[]*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Luca Schalk\"},{\"Id\":4,\"Name\":\"Brad Bullock\"},{\"Id\":8,\"Name\":\"William Tippet\"}]}]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]*TeamData{{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Luca Schalk"},
					{4, "Brad Bullock"},
					{8, "William Tippet"},
				}}}, ta)

				tp, _, _, err := maps.Out[[]any](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte("[{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Breakaway\",\"Players\":[{\"Id\":1,\"Name\":\"Luca Schalk\"},{\"Id\":4,\"Name\":\"Brad Bullock\"},{\"Id\":8,\"Name\":\"William Tippet\"}]}]"), api.FromJson)
				suite.Nil(err)
				suite.Equal([]any{&TeamData{Id: 9, Name: "Breakaway", Players: []PlayerData{
					{1, "Luca Schalk"},
					{4, "Brad Bullock"},
					{8, "William Tippet"},
				}}}, tp)
			})

			suite.Run("FromJsonReaderTyped", func() {
				reader := strings.NewReader("{\"@type\":\"test.TeamData\",\"Id\":9,\"Name\":\"Manchester United\"}")
				data, _, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					reader, api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 9, Name: "Manchester United"}, *data)
			})

			suite.Run("FromJsonReaderTypedLate", func() {
				reader := strings.NewReader("{\"@type\":\"test.TeamData\",\"Id\":11,\"Name\":\"Chelsea\"}")
				late, _, _, err := maps.Out[api.Late](
					miruken.BuildUp(handler, api.Polymorphic),
					reader, api.FromJson)
				suite.Nil(err)
				suite.NotNil(late)
				suite.IsType(&TeamData{}, late.Value)
				suite.Equal(TeamData{Id: 11, Name: "Chelsea"}, *late.Value.(*TeamData))
			})

			suite.Run("FromJsonBytesNoTypeInfo", func() {
				j := "{\"Id\":23,\"Name\":\"Everton\"}"
				data, _, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte(j), api.FromJson)
				suite.Nil(err)
				suite.NotNil(data)
				suite.Equal(TeamData{Id: 23, Name: "Everton"}, *data)
			})

			suite.Run("FromJsonBytesNoTypeInfoAny", func() {
				j := "{\"Id\":19,\"Name\":\"Wolves\"}"
				dat, _, _, err := maps.Out[any](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte(j), api.FromJson)
				suite.Nil(err)
				suite.True(reflect.DeepEqual(map[string]any{
					"Id": float64(19), "Name": "Wolves",
				}, dat))
			})

			suite.Run("FromJsonBytesNoTypeInfoLate", func() {
				j := "{\"Id\":23,\"Name\":\"Everton\"}"
				late, _, _, err := maps.Out[api.Late](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte(j), api.FromJson)
				suite.Nil(err)
				suite.True(reflect.DeepEqual(map[string]any{
					"Id": float64(23), "Name": "Everton",
				}, late.Value))
			})

			suite.Run("FromJsonBytesMissingTypeInfo", func() {
				j := "{\"@type\":\"test.Team\",\"Id\":9,\"Name\":\"Leeds United\"}"
				_, _, _, err := maps.Out[*TeamData](
					miruken.BuildUp(handler, api.Polymorphic),
					[]byte(j), api.FromJson)
				suite.NotNil(err)
			})
		})
	})
}

func TestStdJsonTestSuite(t *testing.T) {
	suite.Run(t, new(StdJsonTestSuite))
}
