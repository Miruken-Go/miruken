package handles

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)

type (
	It      = miruken.Handles
	Strict  = miruken.Strict
	Builder = miruken.HandlesBuilder
)

func Command(
	handler     miruken.Handler,
	callback    any,
	constraints ...any,
) (pv *promise.Promise[any], err error) {
	return miruken.Command(handler, callback, constraints...)
}

func Request[T any](
	handler     miruken.Handler,
	callback    any,
	constraints ...any,
) (t T, tp *promise.Promise[T], err error) {
	return miruken.Execute[T](handler, callback, constraints...)
}

func CommandAll(
	handler     miruken.Handler,
	callback    any,
	constraints ...any,
) (pv *promise.Promise[any], err error) {
	return miruken.CommandAll(handler, callback, constraints...)
}

func RequestAll[T any](
	handler     miruken.Handler,
	callback    any,
	constraints ...any,
) (t []T, tp *promise.Promise[[]T], err error) {
	return miruken.ExecuteAll[T](handler, callback, constraints...)
}