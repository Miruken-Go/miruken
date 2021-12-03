package test

import (
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
		suite.Run("Implicit", func() {
			handler := suite.InferenceRoot()
			entity  := &PlayerEntity{
				Entity{ Id: 1 },
				"Tim Howard",
			}
			var data *PlayerData
			err := miruken.Map(handler, entity, &data)
			suite.Nil(err)
			suite.Equal(1, data.Id)
			suite.Equal("Tim Howard", data.Name)
		})

		suite.Run("Into", func() {
			handler := suite.InferenceRoot()
			entity  := &PlayerEntity{
				Entity{ Id: 2 },
				"David Silva",
			}
			var data PlayerData
			err := miruken.Map(handler, entity, &data)
			suite.Nil(err)
			suite.Equal(2, data.Id)
			suite.Equal("David Silva", data.Name)
		})

		suite.Run("IntoPtr", func() {
			handler := suite.InferenceRoot()
			entity  := &PlayerEntity{
				Entity{ Id: 3 },
				"Franz Beckenbauer",
			}
			data := new(PlayerData)
			err  := miruken.Map(handler, entity, &data)
			suite.Nil(err)
			suite.Equal(3, data.Id)
			suite.Equal("Franz Beckenbauer", data.Name)
		})
	})
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}
