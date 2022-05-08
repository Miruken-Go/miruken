package miruken

import (
	"github.com/miruken-go/miruken/slices"
	"reflect"
)

type (
	// RegistrationBuilder builds an installation.
	RegistrationBuilder struct {
		noInfer      bool
		handlers     []any
		handlerTypes []reflect.Type
		exclude      Predicate[reflect.Type]
		factory      HandlerDescriptorFactory
		tags         map[any]struct{}
	}

	// Installer populates a RegistrationBuilder.
	Installer interface {
		Install(registration *RegistrationBuilder)
	}

	InstallerFunc func(registration *RegistrationBuilder)
)

func (f InstallerFunc) Install(
	registration *RegistrationBuilder,
)  { f(registration) }

func (r *RegistrationBuilder) AddHandlers(
	handlers ... any,
) *RegistrationBuilder {
	r.handlers = append(r.handlers, handlers...)
	return r
}

func (r *RegistrationBuilder) AddHandlerTypes(
	handlerTypes ... reflect.Type,
) *RegistrationBuilder {
	r.handlerTypes = append(r.handlerTypes, handlerTypes...)
	return r
}

func (r *RegistrationBuilder) Exclude(
	excludes ... Predicate[reflect.Type],
) *RegistrationBuilder {
	r.exclude = CombinePredicates(r.exclude, excludes...)
	return r
}

func (r *RegistrationBuilder) AddFilters(
	providers ... FilterProvider,
) *RegistrationBuilder {
	var handles Handles
	handles.Policy().AddFilters(providers...)
	return r
}

func (r *RegistrationBuilder) SetHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) *RegistrationBuilder {
	r.factory = factory
	return r
}

func (r *RegistrationBuilder) DisableInference() {
	r.noInfer = false
}

func (r *RegistrationBuilder) CanInstall(tag any) bool {
	if tags := r.tags; tags == nil {
		r.tags = map[any]struct{} { tag: {} }
		return true
	} else if _, found := tags[tag]; !found {
		tags[tag] = struct{}{}
		return true
	}
	return false
}

func (r *RegistrationBuilder) Install(installer Installer) *RegistrationBuilder {
	installer.Install(r)
	return r
}

func (r *RegistrationBuilder) Build() Handler {
	factory := r.factory
	if IsNil(factory) {
		factory = NewMutableHandlerDescriptorFactory()
	}
	var handler Handler = &getHandlerDescriptorFactory{factory}

	if types := r.handlerTypes; len(types) > 0 {
		if exclude := r.exclude; exclude != nil {
			types = slices.Filter(types, func(t reflect.Type) bool {
				return !exclude(t)
			})
		}
		if len(types) > 0 {
			if r.noInfer {
				for _, typ := range types {
					if _, _, err := factory.RegisterHandlerType(typ); err != nil {
						panic(err)
					}
				}

			} else {
				handler = &withHandler{handler, newInferenceHandler(factory, types)}
			}
		}
	}

	// Handler overrides
	if handlers := r.handlers; len(handlers) > 0 {
		handler = AddHandlers(handler, handlers...)
	}

	return handler
}

var DisableInference = InstallerFunc(func(registration *RegistrationBuilder) {
	registration.noInfer = true
})

func WithHandlers(handlers ... any) Installer {
	return InstallerFunc(func(registration *RegistrationBuilder) {
		registration.AddHandlers(handlers...)
	})
}

func WithHandlerTypes(types ... reflect.Type) Installer {
	return InstallerFunc(func(registration *RegistrationBuilder) {
		registration.AddHandlerTypes(types...)
	})
}

func ExcludeHandlerTypes(filter ... Predicate[reflect.Type]) Installer {
	return InstallerFunc(func(registration *RegistrationBuilder) {
		registration.Exclude(filter...)
	})
}

func WithHandlerDescriptorFactory(factory HandlerDescriptorFactory) Installer {
	return InstallerFunc(func(registration *RegistrationBuilder) {
		registration.SetHandlerDescriptorFactory(factory)
	})
}

func NewRegistration(installers ... Installer) *RegistrationBuilder {
	registration := &RegistrationBuilder{}
	for _, i := range installers {
		if i != nil {
			i.Install(registration)
		}
	}
	return registration
}
