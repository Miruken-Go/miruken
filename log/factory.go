package log

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
)

// factory of context specific loggers.
// The context of the logger will be derived from
//   parent resolution key (constructors) OR
//   resolution owner type (method handlers)
type factory struct {
	root logr.Logger
}

func (f *factory) ContextLogger(
	provides *miruken.Provides,
) (logr.Logger, miruken.HandleResult) {
	var name string
	if parent := provides.Parent(); parent != nil {
		name = fmt.Sprintf("%v", parent.Key())
	} else if owner := provides.Owner(); owner != nil {
		name = fmt.Sprintf("%T", owner)
	}
	if len(name) > 0 {
		return f.root.WithName(name), miruken.Handled
	}
	return f.root, miruken.Handled
}
