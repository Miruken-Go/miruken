package swagger

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/myml/swag/endpoint"
	"github.com/myml/swag/swagger"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// Installer configures swagger support
type Installer struct {
	handlesPolicy miruken.Policy
	endpoints     []*swagger.Endpoint
}

func (i *Installer) DependsOn() []miruken.Feature {
	return []miruken.Feature{httpsrv.Feature()}
}

func (i *Installer) Install(setup *miruken.SetupBuilder) error {
	if setup.Tag(&featureTag) {
		var handles miruken.Handles
		i.handlesPolicy = handles.Policy()
		setup.Observers(i)
	}
	return nil
}

func (i *Installer) Endpoints(handler http.Handler) []*swagger.Endpoint {
	for _, ep := range i.endpoints {
		ep.Handler = handler
	}
	return i.endpoints
}

func (i *Installer) BindingCreated(
	policy     miruken.Policy,
	descriptor *miruken.HandlerDescriptor,
	binding    miruken.Binding,
) {
	if policy != i.handlesPolicy {
		return
	}
	if inputType, ok := binding.Key().(reflect.Type); ok {
		if inputType.Kind() == reflect.Ptr {
			inputType = inputType.Elem()
		}
		if unicode.IsLower([]rune(inputType.Name())[0]) {
			return
		}
		path := strings.Replace(inputType.String(), ".", "/", 1)
		ep := endpoint.New("post", "/process/" + path, "",
			endpoint.BodyType(i.schema(inputType), "request to process", true),
			endpoint.ResponseType(http.StatusOK, i.schema(binding.LogicalOutputType()), "Successfully handled"),
			endpoint.Response(http.StatusInternalServerError, api.ErrorData{}, "Oops ... something went wrong"),
			endpoint.Tags(inputType.PkgPath()),
		)
		i.endpoints = append(i.endpoints, ep)
	}
}

func (i *Installer) DescriptorCreated(
	_ *miruken.HandlerDescriptor,
) {
}

func (i *Installer) schema(typ reflect.Type) reflect.Type {
	if typ == nil {
		return emptySchema
	}
	if anyType.AssignableTo(typ) {
		return emptySchema
	}
	return reflect.StructOf([]reflect.StructField{
		{
			Name: "Payload",
			Type: typ,
			Tag:  `json:"payload"`,
		},
	})
}


// Feature configures http server support
func Feature(
	config ...func(installer *Installer),
) *Installer {
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
	anyType     = miruken.TypeOf[any]()
	emptySchema = miruken.TypeOf[struct{}]()
)
