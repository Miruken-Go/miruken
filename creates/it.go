package creates

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)

type (
	It      = miruken.Creates
	Strict  = miruken.Strict
	Builder = miruken.CreatesBuilder
)

func New[T any](
	handler     miruken.Handler,
	constraints ...any,
) (T, *promise.Promise[T], error) {
	return miruken.Create[T](handler, constraints...)
}

func Key[T any](
	handler     miruken.Handler,
	key         any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	return miruken.CreateKey[T](handler, key, constraints...)
}

func All[T any](
	handler     miruken.Handler,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	return miruken.CreateAll[T](handler, constraints...)
}