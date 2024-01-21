package intent

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
)


type (
	intentGroup struct {
		intents []miruken.Intent
	}
)


func (g *intentGroup) Intents() []miruken.Intent {
	return g.intents
}

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
		x := ps[0]
		return promise.Erase(x), nil
	default:
		x := promise.All(nil, ps...)
		return promise.Erase(x), nil
	}
}


// Group represents a group of intents.
func Group(
	intents ...miruken.Intent,
) miruken.Intent {
	return &intentGroup{intents}
}
