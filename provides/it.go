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
)

func Type[T any](
	handler     miruken.Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return miruken.Resolve[T](handler, constraints...)
}

func Key[T any](
	handler     miruken.Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	return miruken.ResolveKey[T](handler, key, constraints...)
}

func All[T any](
	handler     miruken.Handler,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	return miruken.ResolveAll[T](handler, constraints...)
}