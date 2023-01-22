package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
)

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
	maps   *miruken.Maps,
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
	maps *miruken.Maps, entity *PlayerEntity,
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
	_ *miruken.Maps, entity *PlayerEntity,
) map[string]any {
	return map[string]any{
		"Id":   entity.Id,
		"Name": entity.Name,
	}
}

func (m *EntityMapper) FromPlayerMap(
	_ *miruken.Maps, data map[string]any,
) *PlayerEntity {
	return &PlayerEntity{
		Entity{Id: data["Id"].(int)},
		data["Name"].(string),
	}
}

// OpenMapper
type OpenMapper struct{}

func (m *OpenMapper) Map(
	maps *miruken.Maps,
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
		miruken.Maps
		miruken.Format `to:"application/json"`
	  }, data *PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, data.Name)
}

func (m *FormatMapper) FromPlayerJson(
	_*struct{
		miruken.Maps
		miruken.Format `from:"application/json"`
	  }, jsonString string,
) (PlayerData, error) {
	data := PlayerData{}
	err  := json.Unmarshal([]byte(jsonString), &data)
	return data, err
}

func (m *FormatMapper) StartsWith(
	_*struct{
		miruken.Maps
		miruken.Format `to:"/hello"`
	  }, _ *PlayerData,
) string {
	return "startsWith"
}

func (m *FormatMapper) EndsWith(
	_*struct{
		miruken.Maps
		miruken.Format `to:"world/"`
	  }, _ *PlayerData,
) string {
	return "endsWith"
}

func (m *FormatMapper) Pattern(
	_*struct{
		miruken.Maps
		miruken.Format `to:"/J\\d+![A-Z]+\\d/"`
	  }, _ *PlayerData,
) string {
	return "pattern"
}

// InvalidMapper
type InvalidMapper struct {}

func (m *InvalidMapper) MissingDependency(
	_ *miruken.Handles, _ *Bar,
	_*struct{ },
) {
}

func (m *InvalidMapper) MissingReturnValue(*miruken.Provides) {
}

func (m *InvalidMapper) TooManyReturnValues(
	_ *miruken.Handles, _ *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (m *InvalidMapper) SecondReturnMustBeErrorOrHandleResult(
	_ *miruken.Handles, _ *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}

func (m *InvalidMapper) UntypedInterfaceDependency(
	_ *miruken.Handles, _ *Bar,
	any any,
) miruken.HandleResult {
	return miruken.Handled
}

func (m *InvalidMapper) MissingCallbackArgument(
	_*struct{ miruken.Handles },
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

func (suite *MapsTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup().Specs(suite.specs...).Handler()
}

func (suite *MapsTestSuite) TestMap() {
	suite.Run("Maps", func () {
		suite.Run("New", func() {
			handler, _ := suite.Setup()
			entity := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			data, _, err := miruken.Map[*PlayerData](handler, &entity)
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
			_, err := miruken.MapInto(handler, &entity, &data)
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
			_, err  := miruken.MapInto(handler, &entity, data)
			suite.Nil(err)
			suite.Equal(3, data.Id)
			suite.Equal("Franz Beckenbauer", data.Name)
		})

		suite.Run("Open", func() {
			handler, _ := miruken.Setup().Specs(&OpenMapper{}).Handler()
			entity := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			data, _, err := miruken.Map[*PlayerData](handler, &entity)
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
			data, _, err := miruken.Map[map[string]any](handler, &entity)
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
			entity, _, err := miruken.Map[*PlayerEntity](handler, data)
			suite.Nil(err)
			suite.Equal(2, entity.Id)
			suite.Equal("George Best", entity.Name)
		})

		suite.Run("Format", func() {
			handler, _ := miruken.Setup().Specs(&FormatMapper{}).Handler()

			data  := PlayerData{
				Id:   1,
				Name: "Tim Howard",
			}
			jsonString, _, err := miruken.Map[string](handler, &data, miruken.To("application/json"))
			suite.Nil(err)
			suite.Equal("{\"id\":1,\"name\":\"Tim Howard\"}", jsonString)

			_, _, err = miruken.Map[string](handler, &data, miruken.To("foo"))
			suite.IsType(err, &miruken.NotHandledError{})

			var data2 PlayerData
			_, err = miruken.MapInto(handler, jsonString, &data2, miruken.From("application/json"))
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

			data, _, err := miruken.MapAll[*PlayerData](handler, entities)
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
					if err, ok := r.(*miruken.HandlerDescriptorError); ok {
						var errMethod *miruken.MethodBindingError
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
			_, err := miruken.Setup().Specs(&InvalidMapper{}).Handler()
			suite.Nil(err)
			suite.Fail("should cause panic")
		})
	})

	suite.Run("Format", func () {
		suite.Run("StartsWith", func () {
			handler, _ := miruken.Setup().Specs(&FormatMapper{}).Handler()
			var data PlayerData
			res, _, err := miruken.Map[string](handler, &data, miruken.To("hello"))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("hellohello"))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("/hello"))
			suite.Nil(err)
			suite.Equal("startsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("hel"))
			suite.NotNil(err)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("/hel"))
			suite.NotNil(err)
		})

		suite.Run("EndsWith", func () {
			handler, _ := miruken.Setup().Specs(&FormatMapper{}).Handler()
			var data PlayerData
			res, _, err := miruken.Map[string](handler, &data, miruken.To("world"))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("theworld"))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("world/"))
			suite.Nil(err)
			suite.Equal("endsWith", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("worldwide"))
			suite.NotNil(err)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("wor/"))
			suite.NotNil(err)
		})

		suite.Run("Pattern", func () {
			handler, _ := miruken.Setup().Specs(&FormatMapper{}).Handler()
			var data PlayerData
			res, _, err := miruken.Map[string](handler, &data, miruken.To("J9!P3"))
			suite.Nil(err)
			suite.Equal("pattern", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("J256!ABC1"))
			suite.Nil(err)
			suite.Equal("pattern", res)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("J!2"))
			suite.NotNil(err)
			res, _, err = miruken.Map[string](handler, &data, miruken.To("J85!92"))
			suite.NotNil(err)
		})
	})
}

func TestMapsTestSuite(t *testing.T) {
	suite.Run(t, new(MapsTestSuite))
}
