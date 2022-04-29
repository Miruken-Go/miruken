package miruken

import "reflect"

type (
	// Registration manages the state of an installation.
	Registration struct {
		noInfer      bool
		handlers     []any
		handlerTypes []reflect.Type
		exclude      Predicate[reflect.Type]
		factory      HandlerDescriptorFactory
		tags         map[any]struct{}
	}

	// Installer populates a Registration.
	Installer interface {
		Install(registration *Registration)
	}

	InstallerFunc func(registration *Registration)
)

func (f InstallerFunc) Install(
	registration *Registration,
)  { f(registration) }

func (r *Registration) AddHandlers(
	handlers ... any,
) *Registration {
	r.handlers = append(r.handlers, handlers...)
	return r
}

func (r *Registration) AddHandlerTypes(
	handlerTypes ... reflect.Type,
) *Registration {
	r.handlerTypes = append(r.handlerTypes, handlerTypes...)
	return r
}

func (r *Registration) ExcludeHandlerTypes(
	excludes ... Predicate[reflect.Type],
) *Registration {
	r.exclude = CombinePredicates(r.exclude, excludes...)
	return r
}

func (r *Registration) AddFilters(
	providers ... FilterProvider,
) *Registration {
	var handles Handles
	handles.Policy().AddFilters(providers...)
	return r
}

func (r *Registration) SetHandlerDescriptorFactory(
	factory HandlerDescriptorFactory,
) *Registration {
	r.factory = factory
	return r
}

func (r *Registration) DisableInference() {
	r.noInfer = false
}

func (r *Registration) CanInstall(tag any) bool {
	if tags := r.tags; tags == nil {
		r.tags = map[any]struct{} { tag: {} }
		return true
	} else if _, found := tags[tag]; !found {
		tags[tag] = struct{}{}
		return true
	}
	return false
}

func (r *Registration) Install(installer Installer) *Registration {
	installer.Install(r)
	return r
}

func (r *Registration) Build() Handler {
	factory := r.factory
	if IsNil(factory) {
		factory = NewMutableHandlerDescriptorFactory()
	}
	var handler Handler = &getHandlerDescriptorFactory{factory}

	if types := r.handlerTypes; len(types) > 0 {
		if exclude := r.exclude; exclude != nil {
			types = FilterSlice(types, func(t reflect.Type) bool {
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

var DisableInference = InstallerFunc(func(registration *Registration) {
	registration.noInfer = true
})

func WithHandlers(handlers ... any) Installer {
	return InstallerFunc(func(registration *Registration) {
		registration.AddHandlers(handlers...)
	})
}

func WithHandlerTypes(types ... reflect.Type) Installer {
	return InstallerFunc(func(registration *Registration) {
		registration.AddHandlerTypes(types...)
	})
}

func ExcludeHandlerTypes(filter ... Predicate[reflect.Type]) Installer {
	return InstallerFunc(func(registration *Registration) {
		registration.ExcludeHandlerTypes(filter...)
	})
}

func WithHandlerDescriptorFactory(factory HandlerDescriptorFactory) Installer {
	return InstallerFunc(func(registration *Registration) {
		registration.SetHandlerDescriptorFactory(factory)
	})
}

func NewRegistration(installers ... Installer) *Registration {
	registration := &Registration{}
	for _, i := range installers {
		if i != nil {
			i.Install(registration)
		}
	}
	return registration
}