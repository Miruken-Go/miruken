package provides

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)

type (
	It      = miruken.Provides
	Builder = miruken.ProvidesBuilder
	Single  = miruken.Single
	Strict  = miruken.Strict
	Init    = miruken.Init
)


var (
	With          = miruken.With
	New           = miruken.NewProvider
	WithLifestyle = miruken.UseLifestyle
)



func Type[T any](
	handler     miruken.Handler,
	constraints ...any,
) (T, *promise.Promise[T], bool, error) {
	return miruken.Resolve[T](handler, constraints...)
}

func Key[T any](
	handler     miruken.Handler,
	key         any,
	constraints ...any,
) (T, *promise.Promise[T], bool, error) {
	return miruken.ResolveKey[T](handler, key, constraints...)
}

func All[T any](
	handler     miruken.Handler,
	constraints ...any,
) ([]T, *promise.Promise[[]T], error) {
	return miruken.ResolveAll[T](handler, constraints...)
}
