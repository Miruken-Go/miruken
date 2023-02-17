package openapi

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/maps"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type (
	// Installer configures openapi support
	Installer struct {
		policy         miruken.Policy
		schemas        openapi3.Schemas
		requestBodies  openapi3.RequestBodies
		responses      openapi3.Responses
		paths          openapi3.Paths
		generator      *openapi3gen.Generator
		components     map[reflect.Type]*openapi3.SchemaRef
		typeInfoFormat string
	}

	validationFailure struct {
		PropertyName string
		Errors       []string
		Nested       []validationFailure
	}

	unknownError struct {
		Message string
	}
)


func (i *Installer) Merge(api *openapi3.T)  {
	if api == nil {
		panic("api cannot be nil")
	}
	components := api.Components
	if components == nil {
		components = new(openapi3.Components)
		api.Components = components
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
		if api.Paths == nil {
			api.Paths = make(openapi3.Paths)
		}
		for name, path := range i.paths {
			if _, ok := api.Paths[name]; ok {
				break
			} else {
				api.Paths[name] = path
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
		setup.Observers(i)
		i.addInitialComponents()
	}
	return nil
}

func (i *Installer) AfterInstall(
	setup   *miruken.SetupBuilder,
	handler miruken.Handler,
) error {
	for typ, schema := range i.generator.Types {
		if typ.Kind() == reflect.Struct {
			if _, ok := i.schemas[typ.Name()]; !ok {
				i.schemas[typ.Name()] = schema
			}
		}
	}
	i.generateExampleJson(miruken.BuildUp(handler, api.Polymorphic))
	i.generator = nil
	i.components = nil
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
		if inputType.Kind() != reflect.Struct {
			return
		}
		if _, ok := i.components[inputType]; ok {
			return
		}
		if schema := i.generateSchemaRef(inputType); schema == nil {
			return
		} else {
			inputName := inputType.Name()
			i.components[inputType] = schema
			requestBody := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithDescription("Request to process").
					WithRequired(true).
					WithJSONSchema(openapi3.NewSchema().
						WithPropertyRef("payload", &openapi3.SchemaRef{
						Ref: "#/components/schemas/"+inputName,
					})),
				}
			requestName := inputName+"Request"
			i.requestBodies[requestName] = requestBody

			responseName := "NoResponse"
			if outputType := binding.LogicalOutputType(); outputType != nil {
				if outputType.Kind() == reflect.Ptr {
					outputType = outputType.Elem()
				}
				outputName := outputType.Name()
				if len(outputName) > 0 {
					if _, ok := i.components[outputType]; !ok {
						if schema := i.generateSchemaRef(outputType); schema != nil {
							i.components[outputType] = schema
							response := &openapi3.ResponseRef{
								Value: openapi3.NewResponse().
									WithDescription("Successfully handled").
									WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
										WithPropertyRef("payload", &openapi3.SchemaRef{
											Ref: "#/components/schemas/" + outputName,
										}))),
							}
							responseName = outputName + "Response"
							i.responses[responseName] = response
						}
					} else {
						responseName = outputName + "Response"
					}
				}
			}
			path := &openapi3.PathItem{
				Post: &openapi3.Operation{
					OperationID: inputName,
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
							Ref: "#/components/responses/UnknownError",
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

func (i *Installer) addInitialComponents() {
	for _, extra := range extraTypes {
		_ = i.generateSchemaRef(extra)
	}
	i.responses["NoResponse"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewSchema()))),
	}
	i.responses["ValidationError"] =  &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Validation failed").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithPropertyRef("payload", &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "array",
						Items: &openapi3.SchemaRef{
							Ref: "#/components/schemas/validationFailure",
						},
					},
				}))),
	}
	i.responses["UnknownError"] =  &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Oops ... something went wrong").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithPropertyRef("payload", &openapi3.SchemaRef{
					Ref: "#/components/schemas/unknownError",
				}))),
	}
}

func (i *Installer) generateExampleJson(
	handler miruken.Handler,
) {
	for _, schema := range i.components {
		if example := schema.Value.Example; !miruken.IsNil(example) {
			if reflect.TypeOf(example).Kind() == reflect.Struct {
				if js, _, err := maps.Map[string](handler, example, api.ToJson); err == nil {
					schema.Value.Example = json.RawMessage(js)
				}
			}
		}
	}
}

func (i *Installer) generateSchemaRef(
	t reflect.Type,
) *openapi3.SchemaRef {
	if example := reflect.Zero(t).Interface(); !miruken.IsNil(example) {
		if schema, err := i.generator.NewSchemaRefForValue(example, i.schemas); err == nil {
			schema.Value.Example = example
			return schema
		}
	}
	return nil
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

// TypeInfoFormat selects how the type discriminator is generated.
func TypeInfoFormat(format string ) func(installer *Installer) {
	return func(installer *Installer) {
		installer.typeInfoFormat = format
	}
}

// Feature configures http server support
func Feature(config ...func(installer *Installer)) *Installer {
	installer := &Installer{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var (
	featureTag byte
	extraTypes = []reflect.Type{
		miruken.TypeOf[validationFailure](),
		miruken.TypeOf[unknownError](),
	}
)
