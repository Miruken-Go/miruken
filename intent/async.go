package intent

import (
	"context"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)


type (
	asyncIntent struct {
		intent miruken.Intent
		ctx    context.Context
	}
)


func (a *asyncIntent) Apply(
	ctx miruken.HandleContext,
) (promise.Reflect, error) {
	return promise.New(a.ctx, func(
		resolve func(struct{}), reject func(error), onCancel func(func())) {
		if pi, err := a.intent.Apply(ctx); err != nil {
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


// Async wraps an intent to be executed asynchronously.
func Async(
	ctx    context.Context,
	intent any,
) (miruken.Intent, error) {
	if i, err := Ensure(intent); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else {
		return &asyncIntent{i, ctx}, nil
	}
}


// Ensure ensures the intent is a miruken.Intent.
func Ensure(intent any) (miruken.Intent, error) {
	return miruken.MakeIntent(intent, true)
}
