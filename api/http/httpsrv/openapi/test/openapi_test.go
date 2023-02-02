package test

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/http/httpsrv/openapi"
	"github.com/miruken-go/miruken/api/json/jsonstd"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	CreatePlayer struct {
		Name      string
		BirthDate time.Time
	}

	PlayerResult struct {
		PlayerId int32
	}

	PlayerHandler struct {
		nextId int32
	}
)

func (p *PlayerHandler) CreatePlayer(
	_ *handles.It, create *CreatePlayer,
	ctx miruken.HandleContext,
) *promise.Promise[PlayerResult] {
	id := atomic.AddInt32(&p.nextId,1)
	return promise.Resolve(PlayerResult{id})
}

type OpenApiTestSuite struct {
	suite.Suite
	srv *httptest.Server
}

func (suite *OpenApiTestSuite) Setup(specs ...any) *context.Context {
	handler, _ := miruken.Setup(
		TestFeature, http.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Specs(specs...).
		Handler()
	return context.New(handler)
}

func (suite *OpenApiTestSuite) SetupTest() {
	handler, _ := miruken.Setup(
		TestFeature, openapi.Feature(), jsonstd.Feature()).
		Specs(&api.GoTypeFieldInfoMapper{}).
		Handler()
	suite.srv = httptest.NewServer(httpsrv.Api(context.New(handler)))
}

func (suite *OpenApiTestSuite) TearDownTest() {
	suite.srv.CloseClientConnections()
	suite.srv.Close()
}

func (suite *OpenApiTestSuite) TestOpenApi() {
	suite.Run("Generates OpenApi", func() {
		schemas := make(openapi3.Schemas)
		customizer := openapi3gen.SchemaCustomizer(func(name string, ft reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
			return nil
		})
		schemaRef, err := openapi3gen.NewSchemaRefForValue(&CreatePlayer{}, schemas,
			openapi3gen.UseAllExportedFields(), openapi3gen.ThrowErrorOnCycle(), customizer)
		suite.Nil(err)
		suite.NotNil(schemaRef)
		data, err := json.MarshalIndent(schemaRef, "", "  ")
		fmt.Printf("%s\n", data)
	})
}

func TestOpenApiTestSuite(t *testing.T) {
	suite.Run(t, new(OpenApiTestSuite))
}
