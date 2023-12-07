package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	context2 "github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	Counter interface {
		Count() int
		Inc() int
	}

	Counted struct {
		count int
	}

	Foo struct { Counted }
	Bar struct { Counted }
)

func (c *Counted) Count() int {
	return c.count
}

func (c *Counted) Inc() int {
	c.count++
	return c.count
}

type Entity struct {
	Id int
}

type PlayerEntity struct {
	Entity
	Name string
}

type PlayerData struct {
	Id   int
	Name string
}

// EntityMapper
type EntityMapper struct{}

func (m *EntityMapper) MapPlayerData(
	maps   *maps.It,
	entity *PlayerEntity,
) *PlayerData {
	if data, ok := maps.Target().(**PlayerData); ok && *data != nil {
		(*data).Id   = entity.Id
		(*data).Name = entity.Name
		return *data
	}
	return &PlayerData{
		Id:   entity.Id,
		Name: entity.Name,
	}
}

func (m *EntityMapper) MapIntoPlayerData(
	maps *maps.It, entity *PlayerEntity,
) PlayerData {
	if data, ok := maps.Target().(*PlayerData); ok && data != nil {
		data.Id   = entity.Id
		data.Name = entity.Name
		return *data
	}
	return PlayerData{
		Id:   entity.Id,
		Name: entity.Name,
	}
}

func (m *EntityMapper) ToPlayerMap(
	_ *maps.It, entity *PlayerEntity,
) map[string]any {
	return map[string]any{
		"Id":   entity.Id,
		"Name": entity.Name,
	}
}

func (m *EntityMapper) FromPlayerMap(
	_ *maps.It, data map[string]any,
) *PlayerEntity {
	return &PlayerEntity{
		Entity{Id: data["Id"].(int)},
		data["Name"].(string),
	}
}

// OpenMapper
type OpenMapper struct{}

func (m *OpenMapper) Map(
	maps *maps.It,
) any {
	if entity, ok := maps.Source().(*PlayerEntity); ok {
		if data, ok := maps.Target().(**PlayerData); ok && *data != nil {
			(*data).Id   = entity.Id
			(*data).Name = entity.Name
			return *data
		}
		return &PlayerData{
			Id:   entity.Id,
			Name: entity.Name,
		}
	}
	return nil
}

// FormatMapper
type (
	FormatMapper struct{}
)

func (m *FormatMapper) ToPlayerJson(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, data *PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":%q}", data.Id, data.Name)
}

func (m *FormatMapper) FromPlayerJson(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, jsonString string,
) (PlayerData, error) {
	data := PlayerData{}
	err  := json.Unmarshal([]byte(jsonString), &data)
	return data, err
}

func (m *FormatMapper) StartsWith(
	_*struct{
		maps.It
		maps.Format `to:"/hello"`
	  }, _ *PlayerData,
) string {
	return "startsWith"
}

func (m *FormatMapper) EndsWith(
	_*struct{
		maps.It
		maps.Format `to:"world/"`
	  }, _ *PlayerData,
) string {
	return "endsWith"
}

func (m *FormatMapper) Pattern(
	_*struct{
		maps.It
		maps.Format `to:"/J\\d+![A-Z]+\\d/"`
	  }, _ *PlayerData,
) string {
	return "pattern"
}

// InvalidMapper
type InvalidMapper struct {}

func (m *InvalidMapper) MissingDependency(
	_ *handles.It, _ *Bar,
	_*struct{ },
) {
}

func (m *InvalidMapper) MissingReturnValue(*provides.It) {
}

func (m *InvalidMapper) TooManyReturnValues(
	_ *handles.It, _ *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (m *InvalidMapper) SecondReturnMustBeErrorOrHandleResult(
	_ *handles.It, _ *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}

func (m *InvalidMapper) UntypedInterfaceDependency(
	_ *handles.It, _ *Bar,
	any any,
) miruken.HandleResult {
	return miruken.Handled
}

func (m *InvalidMapper) MissingCallbackArgument(
	_*struct{handles.It},
) miruken.HandleResult {
	return miruken.Handled
}

type MapsTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *MapsTestSuite) SetupTest() {
	suite.specs = []any{
		&EntityMapper{},
	}
}

func (suite *MapsTestSuite) Setup() (*context2.Context, error) {
	return setup.New().Specs(suite.specs...).Context()
}

func (suite *MapsTestSuite) TestMap() {
	suite.Run("It", func () {
		suite.Run("Out", func() {
			handler, _ := suite.Setup()
			entity := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			data, _, _, err := maps.Out[*PlayerData](handler, &entity)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("Into", func() {
			handler, _ := suite.Setup()
			entity := PlayerEntity{
				Entity{ Id: 2 },
				"David Silva",
			}
			var data PlayerData
			_, _, err := maps.Into(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(2, data.Id)
			suite.Equal("David Silva", data.Name)
		})

		suite.Run("IntoPtr", func() {
			handler, _ := suite.Setup()
			entity := PlayerEntity{
				Entity{ Id: 3 },
				"Franz Beckenbauer",
			}
			data := new(PlayerData)
			_, _, err := maps.Into(handler, &entity, data)
			suite.Nil(err)
			suite.Equal(3, data.Id)
			suite.Equal("Franz Beckenbauer", data.Name)
		})

		suite.Run("Open", func() {
			handler, _ := setup.New().Specs(&OpenMapper{}).Context()
			entity := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			data, _, _, err := maps.Out[*PlayerData](handler, &entity)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("ToMap", func() {
			handler, _ := suite.Setup()
			entity := PlayerEntity{
				Entity{ Id: 1 },
				"Marco Royce",
			}
			data, _, _, err := maps.Out[map[string]any](handler, &entity)
			suite.Nil(err)
			suite.Equal(1, data["Id"])
			suite.Equal("Marco Royce", data["Name"])
		})

		suite.Run("FromMap", func() {
			handler, _ := suite.Setup()
			data := map[string]any{
				"Id":    2,
				"Name": "George Best",
			}
			entity, _, _, err := maps.Out[*PlayerEntity](handler, data)
			suite.Nil(err)
			suite.Equal(2, entity.Id)
			suite.Equal("George Best", entity.Name)
		})

		suite.Run("Format", func() {
			handler, _ := setup.New().Specs(&FormatMapper{}).Context()

			data  := PlayerData{
				Id:   1,
				Name: "Tim Howard",
			}
			jsonString, _, _, err := maps.Out[string](handler, &data, api.ToJson)
			suite.Nil(err)
			suite.Equal("{\"id\":1,\"name\":\"Tim Howard\"}", jsonString)

			_, _, _, err = maps.Out[string](handler, &data, maps.To("foo", nil))
			suite.IsType(err, &miruken.NotHandledError{})

			var data2 PlayerData
			_, _, err = maps.Into(handler, jsonString, &data2, api.FromJson)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("All", func() {
			handler, _ := suite.Setup()
			entities := []*PlayerEntity{
				{
					Entity{ Id: 1 },
					"Christian Pulisic",
				},
				{
					Entity{ Id: 2 },
					"Weston Mckennie",
				},
				{
					Entity{ Id: 3 },
					"Josh Sargent",
				},
			}

			data, _, err := maps.All[*PlayerData](handler, entities)
			suite.Nil(err)
			suite.Len(data, len(entities))
			suite.True(reflect.DeepEqual(data, []*PlayerData{
				{
					Id:   1,
					Name: "Christian Pulisic",
				},
				{
					Id: 2,
					Name: "Weston Mckennie",
				},
				{
					Id: 3,
					Name: "Josh Sargent",
				},
			}))
		})

		suite.Run("Invalid", func () {
			failures := 0
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*miruken.HandlerInfoError); ok {
						var errMethod *miruken.MethodBindingError
						for cause := errors.Unwrap(err.Cause);
							errors.As(cause, &errMethod); cause = errors.Unwrap(cause) {
							failures++
						}
						suite.Equal(6, failures)
					} else {
						suite.Fail("Expected HandlerInfoError")
					}
				}
			}()
			_, err := setup.New().Specs(&InvalidMapper{}).Context()
			suite.Nil(err)
			suite.Fail("should cause panic")
		})
	})

	suite.Run("Format", func () {
		suite.Run("StartsWith", func () {
			handler, _ := setup.New().Specs(&FormatMapper{}).Context()
			var data PlayerData
			res, _, _, err := maps.Out[string](handler, &data, maps.To("hello", nil))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("hellohello", nil))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("/hello", nil))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("hel", nil))
			suite.NotNil(err)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("/hel", nil))
			suite.Nil(err)
		})

		suite.Run("EndsWith", func () {
			handler, _ := setup.New().Specs(&FormatMapper{}).Context()
			var data PlayerData
			res, _, _, err := maps.Out[string](handler, &data, maps.To("world", nil))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("theworld", nil))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("world/", nil))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("worldwide", nil))
			suite.NotNil(err)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("wor/", nil))
			suite.NotNil(err)
		})

		suite.Run("Pattern", func () {
			handler, _ := setup.New().Specs(&FormatMapper{}).Context()
			var data PlayerData
			res, _, _, err := maps.Out[string](handler, &data, maps.To("J9!P3", nil))
			suite.Nil(err)
			suite.Equal("pattern", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("J256!ABC1", nil))
			suite.Nil(err)
			suite.Equal("pattern", res)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("J!2", nil))
			suite.NotNil(err)
			res, _, _, err = maps.Out[string](handler, &data, maps.To("J85!92", nil))
			suite.NotNil(err)
		})
	})
}

func TestMapsTestSuite(t *testing.T) {
	suite.Run(t, new(MapsTestSuite))
}
