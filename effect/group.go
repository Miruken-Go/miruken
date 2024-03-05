package effect

import (
	"context"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)


type (
	effectGroup struct {
		effects []miruken.Effect
		ctx     context.Context
	}
)


func (g *effectGroup) Apply(
	ctx  miruken.HandleContext,
) (promise.Reflect, error) {
	var ps []*promise.Promise[any]
	for _, effect := range g.effects {
		if pi, err := effect.Apply(ctx); err != nil {
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


// Group represents a group of effects.
func Group(
	ctx     context.Context,
	effects ...miruken.Effect,
) miruken.Effect {
	return &effectGroup{effects, ctx}
}
