package openapi

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
)

// Installer configures openapi support
type Installer struct {
	handlesPolicy miruken.Policy
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

func (i *Installer) BindingCreated(
	policy     miruken.Policy,
	descriptor *miruken.HandlerDescriptor,
	binding    miruken.Binding,
) {
	if !(policy == i.handlesPolicy  && binding.Exported()) {
		return
	}
	/*
	if inputType, ok := binding.Key().(reflect.Type); ok {
		if inputType.Kind() == reflect.Ptr {
			inputType = inputType.Elem()
		}
		path := strings.Replace(inputType.String(), ".", "/", 1)
		ep := endpoint.New("post", "/process/" + path, "",
			endpoint.BodyType(i.schema(inputType), "request to process", true),
			endpoint.ResponseType(http.StatusOK, i.schema(binding.LogicalOutputType()), "Successfully handled"),
			endpoint.Response(http.StatusInternalServerError, api.ErrorData{}, "Oops ... something went wrong"),
			endpoint.Tags(inputType.PkgPath()),
		)
	}
	 */
}

func (i *Installer) DescriptorCreated(
	_ *miruken.HandlerDescriptor,
) {
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
	anyType     = miruken.TypeOf[any]()
	emptySchema = miruken.TypeOf[struct{}]()
)
