package intent

import (
	"context"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)


type (
	intentGroup struct {
		intents []miruken.Intent
		ctx     context.Context
	}
)


func (g *intentGroup) Apply(
	ctx  miruken.HandleContext,
) (promise.Reflect, error) {
	var ps []*promise.Promise[any]
	for _, intent := range g.intents {
		if pi, err := intent.Apply(ctx); err != nil {
			return nil, err
		} else if pi != nil {
			ps = append(ps, pi.Then(func(data any) any { return data }))
		}
	}
	switch len(ps) {
	case 0:
		return nil, nil
	case 1:
		return ps[0], nil
	default:
		return promise.All(g.ctx, ps...), nil
	}
}


// Group represents a group of intents.
func Group(
	ctx     context.Context,
	intents ...miruken.Intent,
) miruken.Intent {
	return &intentGroup{intents, ctx}
}
