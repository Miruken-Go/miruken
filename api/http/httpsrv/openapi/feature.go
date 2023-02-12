package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/handles"
	"reflect"
	"time"
)

// Installer configures openapi support
type Installer struct {
	policy         miruken.Policy
	schemas        openapi3.Schemas
	requestBodies  openapi3.RequestBodies
	responses      openapi3.Responses
	paths          openapi3.Paths
	generator      *openapi3gen.Generator
	requests       map[reflect.Type]struct{}
	typeInfoFormat string
}

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
		i.schemas = make(openapi3.Schemas)
		i.requestBodies = make(openapi3.RequestBodies)
		i.responses     = make(openapi3.Responses)
		i.paths         = make(openapi3.Paths)
		i.requests      = make(map[reflect.Type]struct{})
		i.generator     = openapi3gen.NewGenerator(
			openapi3gen.UseAllExportedFields(),
			openapi3gen.SchemaCustomizer(i.customize))
		setup.Observers(i)
	}
	return nil
}

func (i *Installer) AfterInstall(
	*miruken.SetupBuilder, miruken.Handler,
) error {
	for _, extra := range extraTypes {
		_, _ = i.generator.GenerateSchemaRef(extra)
	}
	Loop:
	for typ, schema := range i.generator.Types {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if typ.Kind() == reflect.Struct {
			for _, ex := range skipTypes {
				if ex == typ {
					continue Loop
				}
			}
			i.schemas[schema.Ref] = schema
			schema.Ref = ""
		}
	}
	i.responses["NoResponse"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema())),
	}
	i.generator = nil
	i.requests  = nil
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
		if _, ok := i.requests[inputType]; ok {
			return
		}
		if _, err := i.generator.GenerateSchemaRef(inputType); err != nil {
			return
		} else {
			reqSchema   := i.generator.Types[inputType]
			requestBody := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithDescription("Request to process.").
					WithRequired(true).
					WithJSONSchemaRef(openapi3.NewSchemaRef(
						"#/components/schemas/" + reqSchema.Ref, nil)),
				}
			requestName := reqSchema.Ref + "Request"
			i.requestBodies[requestName] = requestBody

			responseName := "NoResponse"
			if outputType := binding.LogicalOutputType(); outputType != nil {
				if outputType.Kind() == reflect.Ptr {
					outputType = outputType.Elem()
				}
				if outputType.Kind() == reflect.Struct && outputType.NumField() > 0 {
					if _, err := i.generator.GenerateSchemaRef(outputType); err != nil {
						return
					} else {
						respSchema := i.generator.Types[outputType]
						response := &openapi3.ResponseRef{
							Value: openapi3.NewResponse().
								WithDescription("Successfully handled.").
								WithContent(openapi3.NewContentWithJSONSchemaRef(
									openapi3.NewSchemaRef("#/components/schemas/"+respSchema.Ref, nil))),
						}
						responseName = respSchema.Ref + "Response"
						i.responses[responseName] = response
					}
				}
			}
			path := &openapi3.PathItem{
				Post: &openapi3.Operation{
					OperationID: reqSchema.Ref,
					RequestBody: &openapi3.RequestBodyRef{
						Ref: "#/components/requestBodies/" + requestName,
					},
					Responses: openapi3.Responses{
						"200": &openapi3.ResponseRef{
							Ref: "#/components/responses/" + responseName,
						},
					},
					Tags: []string{inputType.PkgPath()},
				},
			}
			i.paths["/process/" + reqSchema.Ref] = path
		}
	}
}

func (i *Installer) DescriptorCreated(
	_ *miruken.HandlerDescriptor,
) {
}

func (i *Installer) customize(
	name   string,
	t      reflect.Type,
	tag    reflect.StructTag,
	schema *openapi3.Schema,
) error {
	if props := schema.Properties; props != nil {
		for idx, prop := range props {
			switch prop.Value.Type {
			case "object":
				props[idx] = &openapi3.SchemaRef{
					Ref: "#/components/schemas/" + prop.Ref,
				}
			case "array":
				return nil
			default:
				props[idx].Ref = ""
			}
		}
	}
	return nil
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
	skipTypes  = []reflect.Type{miruken.TypeOf[time.Time]()}
	extraTypes = []reflect.Type{
		miruken.TypeOf[json.OutcomeSurrogate](),
		miruken.TypeOf[json.ErrorSurrogate](),
	}
)
