package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	json2 "github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/api/json/stdjson"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/setup"
)

type (
	// Installer configures openapi support.
	Installer struct {
		base            openapi3.T
		policy          miruken.Policy
		extraComponents []any
		surrogates      map[reflect.Type]any
		apiProfiles     map[string]*apiProfile
		apiDocs         map[string]*openapi3.T
		modules         []*debug.Module
	}

	apiProfile struct {
		module        *debug.Module
		schemas       openapi3.Schemas
		requestBodies openapi3.RequestBodies
		responses     openapi3.ResponseBodies
		paths         *openapi3.Paths
		generator     *openapi3gen.Generator
		components    map[reflect.Type]*openapi3.SchemaRef
	}
)

func (i *Installer) Docs() map[string]*openapi3.T {
	return i.apiDocs
}

func (i *Installer) DependsOn() []setup.Feature {
	return []setup.Feature{httpsrv.Feature()}
}

func (i *Installer) Install(b *setup.Builder) error {
	if b.Tag(&featureTag) {
		var h handles.It
		i.policy = h.Policy()
		i.apiProfiles = make(map[string]*apiProfile)
		b.Observers(i)

		if bi, ok := debug.ReadBuildInfo(); ok {
			if i.modules = bi.Deps; i.modules != nil {
				// sort lexicographically in descending order to
				// match modules with overlapping prefixes.
				sort.Slice(i.modules, func(j, k int) bool {
					return i.modules[j].Path > i.modules[k].Path
				})
			}
		}
	}
	return nil
}

func (i *Installer) AfterInstall(
	_ *setup.Builder,
	handler miruken.Handler,
) error {
	for _, ap := range i.apiProfiles {
		i.generateExampleJson(ap, miruken.BuildUp(handler, api.Polymorphic))
	}
	base := i.base
	base.OpenAPI = "3.0.0"
	if info := base.Info; info == nil {
		base.Info = &openapi3.Info{Version: "0.0.0"}
	} else {
		base.Info.Version = "0.0.0"
	}
	i.apiDocs = make(map[string]*openapi3.T, len(i.apiProfiles))
	for path, ap := range i.apiProfiles {
		doc := base
		info := *doc.Info
		info.Title = info.Title + " (" + filepath.Base(path) + ")"
		if mod := ap.module; mod != nil {
			info.Version = mod.Version
		}
		doc.Info = &info
		components := doc.Components
		if components == nil {
			components = new(openapi3.Components)
			doc.Components = components
		}
		if schemas := ap.schemas; len(schemas) > 0 {
			if components.Schemas == nil {
				components.Schemas = make(openapi3.Schemas)
			}
			for name, schema := range ap.schemas {
				if name == "" {
					continue
				}
				if _, ok := components.Schemas[name]; !ok {
					components.Schemas[name] = schema
				}
			}
		}
		if requestBodies := ap.requestBodies; len(requestBodies) > 0 {
			if components.RequestBodies == nil {
				components.RequestBodies = make(openapi3.RequestBodies)
			}
			for name, reqBody := range ap.requestBodies {
				if _, ok := components.RequestBodies[name]; !ok {
					components.RequestBodies[name] = reqBody
				}
			}
		}
		if responses := ap.responses; len(responses) > 0 {
			if components.Responses == nil {
				components.Responses = make(openapi3.ResponseBodies)
			}
			for name, response := range ap.responses {
				if _, ok := components.Responses[name]; !ok {
					components.Responses[name] = response
				}
			}
		}
		if paths := ap.paths; paths.Len() > 0 {
			if doc.Paths == nil {
				doc.Paths = new(openapi3.Paths)
			}
			for name, path := range ap.paths.Map() {
				if p := doc.Paths.Value(name); p == nil {
					doc.Paths.Set(name, path)
				}
			}
		}
		i.apiDocs[path] = &doc
	}
	i.apiProfiles = nil
	i.modules = nil
	return nil
}

func (i *Installer) BindingCreated(
	policy miruken.Policy,
	handlerInfo *miruken.HandlerInfo,
	binding miruken.Binding,
) {
	if !(policy == i.policy && binding.Exported()) {
		return
	}
	if inType, ok := binding.Key().(reflect.Type); ok {
		if inType.Kind() == reflect.Ptr {
			inType = inType.Elem()
		}
		spec := handlerInfo.Spec()
		ap := i.apiProfile(spec.PkgPath())
		if schema, inputName, created := i.generateTypeSchema(ap, inType, false); created {
			requestBody := &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithDescription("Request to process").
					WithRequired(true).
					WithJSONSchema(openapi3.NewSchema().
						WithPropertyRef("payload", schema)),
			}
			requestName := inputName + "Request"
			ap.requestBodies[requestName] = requestBody

			responseName := "NoResponse"
			if outType := binding.LogicalOutputType(); outType != nil {
				if outType.Kind() == reflect.Ptr {
					outType = outType.Elem()
				}
				if schema, _, _ := i.generateTypeSchema(ap, outType, true); schema != nil {
					response := &openapi3.ResponseRef{
						Value: openapi3.NewResponse().
							WithDescription("Successful Response").
							WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
								WithPropertyRef("payload", schema))),
					}
					responseName = inputName + "Response"
					ap.responses[responseName] = response
				}
			}
			path := &openapi3.PathItem{
				Post: &openapi3.Operation{
					OperationID: inputName,
					Description: fmt.Sprintf("Handled by %s", spec),
					RequestBody: &openapi3.RequestBodyRef{
						Ref: "#/components/requestBodies/" + requestName,
					},
					Responses: openapi3.NewResponses(
						openapi3.WithStatus(http.StatusOK, &openapi3.ResponseRef{
							Ref: "#/components/responses/" + responseName,
						}),
						openapi3.WithStatus(http.StatusUnprocessableEntity, &openapi3.ResponseRef{
							Ref: "#/components/responses/ValidationError",
						}),
						openapi3.WithStatus(http.StatusUnauthorized, &openapi3.ResponseRef{
							Ref: "#/components/responses/UnauthorizedError",
						}),
						openapi3.WithStatus(http.StatusForbidden, &openapi3.ResponseRef{
							Ref: "#/components/responses/ForbiddenError",
						}),
						openapi3.WithStatus(http.StatusInternalServerError, &openapi3.ResponseRef{
							Ref: "#/components/responses/GenericError",
						}),
					),
					Tags: []string{inType.PkgPath()},
				},
			}
			ap.paths.Set("/process/"+strings.ToLower(inputName), path)
		}
	}
}

func (i *Installer) HandlerInfoCreated(
	_ *miruken.HandlerInfo,
) {
}

func (i *Installer) initializeDefinitions(ap *apiProfile) {
	for _, component := range i.extraComponents {
		schema, name, created := i.generateComponentSchema(ap, component, true)
		if created {
			if _, ok := ap.schemas[name]; !ok {
				ap.schemas[name] = schema
			}
		}
	}
	ap.responses["NoResponse"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Empty Response").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewObjectSchema()))),
	}
	ap.responses["UnauthorizedError"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Unauthorized").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewObjectSchema()))),
	}
	ap.responses["ForbiddenError"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Forbidden").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithProperty("payload", openapi3.NewObjectSchema()))),
	}
	ap.responses["ValidationError"] = &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Validation Error").
			WithContent(openapi3.NewContentWithJSONSchema(openapi3.NewSchema().
				WithPropertyRef("payload", &openapi3.SchemaRef{
					Ref: "#/components/schemas/Outcome",
				}))),
	}
	ap.responses["GenericError"] = &openapi3.ResponseRef{
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
	tags := []string{internal.TypeOf[api.Message]().PkgPath()}
	ap.paths.Set("/process", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "process",
			RequestBody: payload,
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(http.StatusOK, &openapi3.ResponseRef{
					Ref: "#/components/responses/NoResponse",
				}),
				openapi3.WithStatus(http.StatusUnprocessableEntity, &openapi3.ResponseRef{
					Ref: "#/components/responses/ValidationError",
				}),
				openapi3.WithStatus(http.StatusUnauthorized, &openapi3.ResponseRef{
					Ref: "#/components/responses/UnauthorizedError",
				}),
				openapi3.WithStatus(http.StatusForbidden, &openapi3.ResponseRef{
					Ref: "#/components/responses/ForbiddenError",
				}),
				openapi3.WithStatus(http.StatusInternalServerError, &openapi3.ResponseRef{
					Ref: "#/components/responses/GenericError",
				}),
			),
			Tags: tags,
		},
	})
	ap.paths.Set("/publish", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "publish",
			RequestBody: payload,
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(http.StatusOK, &openapi3.ResponseRef{
					Ref: "#/components/responses/NoResponse",
				}),
				openapi3.WithStatus(http.StatusUnprocessableEntity, &openapi3.ResponseRef{
					Ref: "#/components/responses/ValidationError",
				}),
				openapi3.WithStatus(http.StatusUnauthorized, &openapi3.ResponseRef{
					Ref: "#/components/responses/UnauthorizedError",
				}),
				openapi3.WithStatus(http.StatusForbidden, &openapi3.ResponseRef{
					Ref: "#/components/responses/ForbiddenError",
				}),
				openapi3.WithStatus(http.StatusInternalServerError, &openapi3.ResponseRef{
					Ref: "#/components/responses/GenericError",
				}),
			),
			Tags: tags,
		},
	})
}

func (i *Installer) apiProfile(
	pkgPath string,
) *apiProfile {
	if pkgPath == "" {
		return nil
	}
	var module *debug.Module
	for _, mod := range i.modules {
		if strings.HasPrefix(pkgPath, mod.Path) {
			module = mod
			break
		}
	}
	var path string
	if module == nil {
		path = "anonymous"
	} else {
		path = module.Path
	}
	ap, ok := i.apiProfiles[path]
	if !ok {
		ap = &apiProfile{
			module:        module,
			schemas:       make(openapi3.Schemas),
			requestBodies: make(openapi3.RequestBodies),
			responses:     make(openapi3.ResponseBodies),
			paths:         new(openapi3.Paths),
			components:    make(map[reflect.Type]*openapi3.SchemaRef),
		}
		ap.generator = openapi3gen.NewGenerator(
			openapi3gen.SchemaCustomizer(ap.customize),
			openapi3gen.UseAllExportedFields())
		i.initializeDefinitions(ap)
		i.apiProfiles[path] = ap
	}
	return ap
}

func (i *Installer) generateExampleJson(
	ap *apiProfile,
	handler miruken.Handler,
) {
	for _, schema := range ap.components {
		if example := schema.Value.Example; !internal.IsNil(example) {
			if b, _, _, err := maps.Out[[]byte](handler, example, api.ToJson); err == nil {
				var js map[string]any
				if err := json.Unmarshal(b, &js); err == nil {
					if typ, ok := js["@type"]; ok {
						props := schema.Value.Properties
						if props == nil {
							props = openapi3.Schemas{}
						}
						props["@type"] = &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type:    "string",
								Default: typ,
							}}
					}
				}
				schema.Value.Example = json.RawMessage(b)
			}
		}
	}
}

func (i *Installer) generateTypeSchema(
	ma *apiProfile,
	typ reflect.Type,
	shared bool,
) (*openapi3.SchemaRef, string, bool) {
	var component reflect.Value
	switch typ.Kind() {
	case reflect.Slice:
		component = reflect.MakeSlice(typ, 0, 0)
	default:
		component = reflect.Zero(typ)
	}
	return i.generateComponentSchema(ma, component.Interface(), shared)
}

func (i *Installer) generateComponentSchema(
	ap *apiProfile,
	component any,
	shared bool,
) (*openapi3.SchemaRef, string, bool) {
	if internal.IsNil(component) {
		return nil, "", false
	}
	typ := reflect.TypeOf(component)
	if sur, ok := i.surrogates[typ]; ok {
		component = sur
	}
	name := typ.Name()
	kind := typ.Kind()
	list := kind == reflect.Slice || kind == reflect.Array
	var elemTyp reflect.Type
	if list {
		elemTyp = typ.Elem()
		if elemTyp.Kind() == reflect.Ptr {
			elemTyp = elemTyp.Elem()
		}
	}
	if len(name) == 0 {
		if list {
			if name = elemTyp.Name(); len(name) > 0 {
				name = name + "Array"
			}
		}
		if len(name) == 0 {
			return nil, "", false
		}
	}
	name = i.uniqueName(ap, name)
	if schema, ok := ap.components[typ]; !ok {
		var err error
		schema, err = ap.generator.NewSchemaRefForValue(component, ap.schemas)
		if err == nil {
			if list {
				if es, _, _ := i.generateTypeSchema(ap, elemTyp, true); es != nil {
					schema.Value.Items = es
				}
				schema = &openapi3.SchemaRef{
					Value: openapi3.NewObjectSchema().
						WithPropertyRef("@values", schema),
				}
			}
			schema.Value.Example = component
			schemaRef := &openapi3.SchemaRef{
				Ref:   "#/components/schemas/" + name,
				Value: schema.Value,
			}
			ap.components[typ] = schemaRef
			if shared {
				ap.schemas[name] = schema
				schema = schemaRef
			}
			return schema, name, true
		}
		return nil, "", false
	} else {
		return schema, name, false
	}
}

func (i *Installer) uniqueName(
	ap *apiProfile,
	name string,
) string {
	id := 0
	var next = name
	for {
		if _, ok := ap.schemas[next]; !ok {
			return next
		}
		id += 1
		next = name + strconv.Itoa(id)
	}
}

func (ap *apiProfile) customize(
	name string,
	typ reflect.Type,
	tag reflect.StructTag,
	schema *openapi3.Schema,
) error {
	if props := schema.Properties; props != nil {
		for key, scr := range props {
			sc := scr.Value
			// Fix anonymous self-referencing array
			if sc.Type == "array" &&
				sc.Items.Value == schema &&
				sc.Items.Ref == "#/components/schemas/" {
				sn := "schema" + strconv.Itoa(len(ap.schemas))
				sc.Items.Ref = "#/components/schemas/" + sn
				ap.schemas[sn] = &openapi3.SchemaRef{Value: schema}
			}
			camel := camelcase(key)
			if camel != key {
				if _, ok := props[camel]; !ok {
					props[camel] = scr
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
func ExtraComponents(components ...any) func(*Installer) {
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
func Feature(
	base openapi3.T,
	config ...func(*Installer),
) *Installer {
	installer := &Installer{
		base: base,
		extraComponents: []any{
			json2.Outcome{
				{
					PropertyName: "PropertyName",
					Errors: []string{
						"Key: 'PropertyName' Error:Field validation for 'PropertyName' failed on the 'required' tag",
					},
				},
			},
			json2.Error{
				Message: "Something bad happened.",
			},
		},
		surrogates: map[reflect.Type]any{
			internal.TypeOf[api.ConcurrentBatch](): json2.Concurrent{},
			internal.TypeOf[api.SequentialBatch](): json2.Sequential{},
			internal.TypeOf[api.ScheduledResult](): stdjson.ScheduledResult{
				stdjson.Either[error, any]{
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
