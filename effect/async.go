package effect

import (
	"context"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)


type (
	asyncEffect struct {
		effect miruken.Effect
		ctx    context.Context
	}
)


func (a *asyncEffect) Apply(
	ctx miruken.HandleContext,
) (promise.Reflect, error) {
	return promise.New(a.ctx, func(
		resolve func(struct{}), reject func(error), onCancel func(func())) {
		if pi, err := a.effect.Apply(ctx); err != nil {
			reject(err)
		} else if pi != nil {
			if _, err := pi.AwaitAny(); err != nil {
				reject(err)
			} else {
				resolve(struct{}{})
			}
		} else {
			resolve(struct{}{})
		}
	}), nil
}


// Async wraps an effect to be executed asynchronously.
func Async(
	ctx    context.Context,
	effect any,
) (miruken.Effect, error) {
	if i, err := Ensure(effect); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else {
		return &asyncEffect{i, ctx}, nil
	}
}

// Ensure ensures the effect is a miruken.Effect.
func Ensure(effect any) (miruken.Effect, error) {
	return miruken.MakeEffect(effect, true)
}
