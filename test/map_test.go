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
	_*miruken.Maps, entity *PlayerEntity,
) map[string]any {
	return map[string]any{
		"Id":   entity.Id,
		"Name": entity.Name,
	}
}

func (m *EntityMapper) FromPlayerMap(
	_*miruken.Maps, data map[string]any,
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
		miruken.Format `as:"application/json"`
	  }, data *PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, data.Name)
}

func (m *FormatMapper) FromPlayerJson(
	_*struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, jsonString string,
) (PlayerData, error) {
	data := PlayerData{}
	err  := json.Unmarshal([]byte(jsonString), &data)
	return data, err
}

// InvalidMapper
type InvalidMapper struct {}

func (m *InvalidMapper) MissingDependency(
	_*miruken.Handles, _ *Bar,
	_*struct{ },
) {
}

func (m *InvalidMapper) MissingReturnValue(*miruken.Provides) {
}

func (m *InvalidMapper) TooManyReturnValues(
	_*miruken.Handles, _ *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (m *InvalidMapper) SecondReturnMustBeErrorOrHandleResult(
	_*miruken.Handles, _ *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}

func (m *InvalidMapper) UntypedInterfaceDependency(
	_*miruken.Handles, _ *Bar,
	any any,
) miruken.HandleResult {
	return miruken.Handled
}

func (m *InvalidMapper) MissingCallbackArgument(
	_*struct{ miruken.Handles },
) miruken.HandleResult {
	return miruken.Handled
}

type MapTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *MapTestSuite) SetupTest() {
	suite.specs = []any{
		&EntityMapper{},
	}
}

func (suite *MapTestSuite) Setup() miruken.Handler {
	return suite.SetupWith(suite.specs...)
}

func (suite *MapTestSuite) SetupWith(specs ... any) miruken.Handler {
	return miruken.Setup(miruken.WithHandlerSpecs(specs...))
}

func (suite *MapTestSuite) TestMap() {
	suite.Run("Maps", func () {
		suite.Run("New", func() {
			handler := suite.Setup()
			entity  := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			var data *PlayerData
			err := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("Into", func() {
			handler := suite.Setup()
			entity  := PlayerEntity{
				Entity{ Id: 2 },
				"David Silva",
			}
			var data PlayerData
			err := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(2, data.Id)
			suite.Equal("David Silva", data.Name)
		})

		suite.Run("IntoPtr", func() {
			handler := suite.Setup()
			entity  := PlayerEntity{
				Entity{ Id: 3 },
				"Franz Beckenbauer",
			}
			data := new(PlayerData)
			err  := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(3, data.Id)
			suite.Equal("Franz Beckenbauer", data.Name)
		})

		suite.Run("Open", func() {
			handler := suite.SetupWith(&OpenMapper{})
			entity  := PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			var data *PlayerData
			err := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("ToMap", func() {
			handler := suite.Setup()
			entity  := PlayerEntity{
				Entity{ Id: 1 },
				"Marco Royce",
			}
			var data map[string]any
			err := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(1, data["Id"])
			suite.Equal("Marco Royce", data["Name"])
		})

		suite.Run("FromMap", func() {
			handler := suite.Setup()
			data    := map[string]any{
				"Id":    2,
				"Name": "George Best",
			}
			var entity *PlayerEntity
			err := miruken.Map(handler, data, &entity)
			suite.Nil(err)
			suite.Equal(2, entity.Id)
			suite.Equal("George Best", entity.Name)
		})

		suite.Run("Format", func() {
			handler := suite.SetupWith(&FormatMapper{})

			data  := PlayerData{
				Id:   1,
				Name: "Tim Howard",
			}
			var jsonString string
			err := miruken.Map(handler, &data, &jsonString, "application/json")
			suite.Nil(err)
			suite.Equal("{\"id\":1,\"name\":\"Tim Howard\"}", jsonString)

			err = miruken.Map(handler, &data, &jsonString, "foo")
			suite.Error(miruken.NotHandledError{}, err)

			var data2 PlayerData
			err = miruken.Map(handler, jsonString, &data2, "application/json")
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("All", func() {
			handler  := suite.Setup()
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

			var data []*PlayerData
			err := miruken.MapAll(handler, entities, &data)
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
			suite.SetupWith(&InvalidMapper{})
			suite.Fail("should cause panic")
		})
	})
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}
