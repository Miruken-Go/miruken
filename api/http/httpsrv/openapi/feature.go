package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	json2 "github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/api/json/jsonstd"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/maps"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type (
	// Installer configures openapi support.
	Installer struct {
		policy          miruken.Policy
		schemas         openapi3.Schemas
		requestBodies   openapi3.RequestBodies
		responses       openapi3.Responses
		paths           openapi3.Paths
		generator       *openapi3gen.Generator
		components      map[reflect.Type]*openapi3.SchemaRef
		extraComponents []any
		surrogates      map[reflect.Type]any
	}

	// ValidationFailure models a single property failure.
	ValidationFailure struct {
		PropertyName string
		Errors       []string
		Nested       []ValidationFailure
	}
)


func (i *Installer) Merge(docs *openapi3.T)  {
	if docs == nil {
		panic("docs cannot be nil")
	}
	components := docs.Components
	if components == nil {
		components = new(openapi3.Components)
		docs.Components = components
	}
	if schemas := i.schemas; len(schemas) > 0 {
		if components.Schemas == nil {
			components.Schemas = make(openapi3.Schemas)
		}
		for name, schema := range i.schemas {
			if _, ok := components.Schemas[name]; ok {
				break
			} else {
				components.Schemas[name] = schema
			}
		}
	}
	if requestBodies := i.requestBodies; len(requestBodies) > 0 {
		if components.RequestBodies == nil {
			components.RequestBodies = make(openapi3.RequestBodies)
		}
		for name, reqBody := range i.requestBodies {
			if _, ok := components.RequestBodies[name]; ok {
				break
			} else {
				components.RequestBodies[name] = reqBody
			}
		}
	}
	if responses := i.responses; len(responses) > 0 {
		if components.Responses == nil {
			components.Responses = make(openapi3.Responses)
		}
		for name, response := range i.responses {
			if _, ok := components.Responses[name]; ok {
				break
			} else {
				components.Responses[name] = response
			}
		}
	}
	if paths := i.paths; len(paths) > 0 {
		if docs.Paths == nil {
			docs.Paths = make(openapi3.Paths)
		}
		for name, path := range i.paths {
			if _, ok := docs.Paths[name]; ok {
				break
			} else {
				docs.Paths[name] = path
			}
		}
	}
}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{httpsrv.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		var h handles.It
		i.policy        = h.Policy()
		i.schemas       = make(openapi3.Schemas)
		i.requestBodies = make(openapi3.RequestBodies)
		i.responses     = make(openapi3.Responses)
		i.paths         = make(openapi3.Paths)
		i.components = make(map[reflect.Type]*openapi3.SchemaRef)
		i.generator     = openapi3gen.NewGenerator(
			openapi3gen.SchemaCustomizer(i.customize),
			openapi3gen.UseAllExportedFields())
		i.initializeDefinitions()
		setup.Observers(i)
	}
	return nil
}

func (i *Installer) AfterInstall(
	setup   *miruken.SetupBuilder,
	handler miruken.Handler,
) error {
	i.generateExampleJson(miruken.BuildUp(handler, api.Polymorphic))
	return nil
}

func (i *Installer) BindingCreated(
	policy     miruken.Policy,
	descriptor *miruken.HandlerDescriptor,
	binding    miruken.Binding,
) {
	if !(policy == i.policy && binding.Exported()) {
		return
	}
	if inputType, ok := binding.Key().(reflect.Type); ok {
		if inputType.Kind() == reflect.Ptr {
			inputType = inputType.Elem()
		}
		if schema, inputName, created := i.generateTypeSchema(inputType); created {
			requestBody := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithDescription("Request to process").
					WithRequired(true).
					WithJSONSchema(openapi3.NewSchema().
						WithPropertyRef("payload", schema)),
				}
			requestName := inputName+"Request"
			i.requestBodies[requestName] = requestBody

			responseName := "NoResponse"
			if outputType := binding.LogicalOutputType(); outputType != nil {
				if outputType.Kind() == reflect.Ptr {
					outputType = outputType.Elem()
				}
				schema, _, _ := i.generateTypeSchema(outputType)
				response := &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Successful Response").
						WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
							WithPropertyRef("payload", schema))),
				}
				responseName = inputName+"Response"
				i.responses[responseName] = response
			}
			path := &openapi3.PathItem{
				Post: &openapi3.Operation{
					OperationID: inputName,
					Description: fmt.Sprintf("Handled by %s", descriptor.HandlerSpec()),
					RequestBody: &openapi3.RequestBodyRef{
						Ref: "#/components/requestBodies/"+requestName,
					},
					Responses: openapi3.Responses{
						"200": &openapi3.ResponseRef{
							Ref: "#/components/responses/"+responseName,
						},
						"422": &openapi3.ResponseRef{
							Ref: "#/components/responses/ValidationError",
						},
						"500": &openapi3.ResponseRef{
							Ref: "#/components/responses/GenericError",
						},
					},
					Tags: []string{inputType.PkgPath()},
				},
			}
			i.paths["/process/"+strings.ToLower(inputName)] = path
		}
	}
}

func (i *Installer) DescriptorCreated(
	_ *miruken.HandlerDescriptor,
) {
}

func (i *Installer) initializeDefinitions() {
	for _, component := range i.extraComponents {
		schema, name, created := i.generateComponentSchema(component)
		if created {
			if _, ok := i.schemas[name]; !ok {
				i.schemas[name] = schema
			}
		}
	}
	i.responses["NoResponse"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewObjectSchema()))),
	}
	i.responses["ValidationError"] =  &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Validation Error").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithPropertyRef("payload", &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "array",
						Items: &openapi3.SchemaRef{
							Ref: "#/components/schemas/ValidationFailure",
						},
						Example: json.RawMessage(`{
    "@type": "json.Outcome",
    "@values": [
      {
        "propertyName": "PropertyName",
        "errors": [
          "Key: 'PropertyName' Error:Field validation for 'PropertyName' failed on the 'required' tag"
        ],
        "nested": null
      }
    ]
  }`),
					},
				}))),
	}
	i.responses["GenericError"] =  &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Oops ... something went wrong").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithPropertyRef("payload", &openapi3.SchemaRef{
					Ref: "#/components/schemas/Error",
				}))),
	}
	payload := &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithDescription("Request to process").
			WithRequired(true).
			WithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewObjectSchema())),
	}
	tags := []string{miruken.TypeOf[api.Message]().PkgPath()}
	i.paths["/process"] = &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "process",
			RequestBody: payload,
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: "#/components/responses/NoResponse",
				},
				"422": &openapi3.ResponseRef{
					Ref: "#/components/responses/ValidationError",
				},
				"500": &openapi3.ResponseRef{
					Ref: "#/components/responses/GenericError",
				},
			},
			Tags: tags,
		},
	}
	i.paths["/publish"] = &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "publish",
			RequestBody: payload,
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: "#/components/responses/NoResponse",
				},
				"422": &openapi3.ResponseRef{
					Ref: "#/components/responses/ValidationError",
				},
				"500": &openapi3.ResponseRef{
					Ref: "#/components/responses/GenericError",
				},
			},
			Tags: tags,
		},
	}
}

func (i *Installer) generateExampleJson(
	handler miruken.Handler,
) {
	for _, schema := range i.components {
		if example := schema.Value.Example; !miruken.IsNil(example) {
			if b, _, err := maps.Out[[]byte](handler, example, api.ToJson); err == nil {
				schema.Value.Example = json.RawMessage(b)
			}
		}
	}
}

func (i *Installer) generateTypeSchema(
	typ reflect.Type,
) (*openapi3.SchemaRef, string, bool) {
	var component reflect.Value
	switch typ.Kind() {
	case reflect.Slice:
		component = reflect.MakeSlice(typ, 0, 0)
	default:
		component = reflect.Zero(typ)
	}
	return i.generateComponentSchema(component.Interface())
}

func (i *Installer) generateComponentSchema(
	component any,
) (*openapi3.SchemaRef, string, bool) {
	if miruken.IsNil(component) {
		return nil, "", false
	}
	typ  := reflect.TypeOf(component)
	if sur, ok := i.surrogates[typ]; ok {
		component = sur
	}
	name := typ.Name()
	if len(name) == 0 {
		if typ.Kind() == reflect.Slice {
			elem := typ.Elem()
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if name = elem.Name(); len(name) > 0 {
				name = name + "Array"
			}
		}
		if len(name) == 0 {
			return nil, "", false
		}
	}
	if schema, ok := i.components[typ]; !ok {
		var err error
		schema, err = i.generator.NewSchemaRefForValue(component, i.schemas)
		if err == nil {
			schema.Value.Example = component
			i.components[typ] = schema
			return schema, name, true
		}
		return nil, "", false
	} else {
		return schema, name, false
	}
}

func (i *Installer) customize(
	name   string,
	t      reflect.Type,
	tag    reflect.StructTag,
	schema *openapi3.Schema,
) error {
	if props := schema.Properties; props != nil {
		for key, sc := range props {
			camel := camelcase(key)
			if camel != key {
				if _, ok := props[camel]; !ok {
					props[camel] = sc
					delete(props, key)
				}
			}
		}
	}
	return nil
}

func camelcase(name string) string {
	r, n := utf8.DecodeRuneInString(name)
	return string(unicode.ToLower(r)) + name[n:]
}

// ExtraComponents provides additional components to include schemas for.
func ExtraComponents(components ... any) func(*Installer) {
	return func(installer *Installer) {
		installer.extraComponents = append(installer.extraComponents, components...)
	}
}

func Surrogates(surrogates map[reflect.Type]any) func(*Installer) {
	return func(installer *Installer) {
		for typ, example := range surrogates {
			if typ != nil && example != nil {
				installer.surrogates[typ] = example
			}
		}
	}
}

// Feature configures http server support
func Feature(config ...func(installer *Installer)) *Installer {
	installer := &Installer{
		extraComponents: []any{
			ValidationFailure{
				PropertyName: "PropertyName",
				Errors: []string{
					"Key: 'PropertyName' Error:Field validation for 'PropertyName' failed on the 'required' tag",
				},
			},
			json2.Error{
				Message: "Something bad happened.",
			},
		},
		surrogates: map[reflect.Type]any{
			miruken.TypeOf[api.ConcurrentBatch](): json2.Concurrent{},
			miruken.TypeOf[api.SequentialBatch](): json2.Sequential{},
			miruken.TypeOf[api.ScheduledResult](): jsonstd.ScheduledResult{
				jsonstd.Either[error, any]{
					Left:  false,
					Value: json.RawMessage("\"success\""),
				},
			},
		},
	}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var featureTag byte

