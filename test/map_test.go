package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"miruken.com/miruken"
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

type MapTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

// EntityMapper
type EntityMapper struct{}

func (m *EntityMapper) MapPlayerData(
	maps *miruken.Maps, entity *PlayerEntity,
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
) map[string]interface{} {
	return map[string]interface{}{
		"Id":   entity.Id,
		"Name": entity.Name,
	}
}

func (m *EntityMapper) FromPlayerMap(
	_ *miruken.Maps, data map[string]interface{},
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
) interface{} {
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

	JsonOptions struct{
		Prefix string
		Indent string
	}
)

func (m *FormatMapper) ToPlayerJson(
	_ *struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, data *PlayerData,
) string {
	return fmt.Sprintf("{\"id\":%v,\"name\":\"%v\"}", data.Id, data.Name)
}

func (m *FormatMapper) FromPlayerJson(
	_ *struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, jsonString string,
) (PlayerData, error) {
	data := PlayerData{}
	err  := json.Unmarshal([]byte(jsonString), &data)
	return data, err
}

func (m *FormatMapper) ToJson(
	_ *struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_ *struct{
		miruken.Optional
		miruken.FromOptions
	  }, options JsonOptions,
) (js string, err error) {
	var data []byte
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(maps.Source(), prefix, indent)
	} else {
		data, err = json.Marshal(maps.Source())
	}
	return string(data), err
}

// InvalidMapper
type InvalidMapper struct {}

func (h *InvalidMapper) MissingDependency(
	_ *miruken.Handles, _ *Bar,
	_ *struct{ },
) {
}

func (p *InvalidMapper) MissingReturnValue(*miruken.Provides) {
}

func (h *InvalidMapper) TooManyReturnValues(
	_ *miruken.Handles, _ *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (h *InvalidMapper) SecondReturnMustBeErrorOrHandleResult(
	_ *miruken.Handles, _ *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}

func (h *InvalidMapper) UntypedInterfaceDependency(
	_ *miruken.Handles, _ *Bar,
	any interface{},
) miruken.HandleResult {
	return miruken.Handled
}

func (h *InvalidMapper) MissingCallbackArgument(
	_ *struct{ miruken.Handles },
) miruken.HandleResult {
	return miruken.Handled
}

func (suite *MapTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		reflect.TypeOf((*EntityMapper)(nil)),
	}
	suite.HandleTypes = handleTypes
}

func (suite *MapTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *MapTestSuite) InferenceRootWith(
	handlerTypes ... reflect.Type) miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(handlerTypes...))
}

func (suite *MapTestSuite) TestMap() {
	suite.Run("Maps", func () {
		suite.Run("New", func() {
			handler := suite.InferenceRoot()
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
			handler := suite.InferenceRoot()
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
			handler := suite.InferenceRoot()
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
			handler := miruken.NewRootHandler(
				miruken.WithHandlerTypes(reflect.TypeOf((*OpenMapper)(nil))))
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
			handler := suite.InferenceRoot()
			entity  := PlayerEntity{
				Entity{ Id: 1 },
				"Marco Royce",
			}
			var data map[string]interface{}
			err := miruken.Map(handler, &entity, &data)
			suite.Nil(err)
			suite.Equal(1, data["Id"])
			suite.Equal("Marco Royce", data["Name"])
		})

		suite.Run("FromMap", func() {
			handler := suite.InferenceRoot()
			data    := map[string]interface{}{
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
			handler := miruken.NewRootHandler(
				miruken.WithHandlerTypes(reflect.TypeOf((*FormatMapper)(nil))))

			suite.Run("Implicit", func() {
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

			suite.Run("Open", func() {
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

			suite.Run("Options", func() {
				data := struct{
					Name string
					Age  int
				}{
					"Sarah Conner",
					38,
				}
				var jsonString string
				err := miruken.Map(
					miruken.Build(handler, miruken.WithOptions(JsonOptions{Indent: "  "})),
					data, &jsonString, "application/json")
				suite.Nil(err)
				suite.Equal("{\n  \"Name\": \"Sarah Conner\",\n  \"Age\": 38\n}", jsonString)
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

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}
