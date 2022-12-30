package log

import (
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"reflect"
)

// factory of context specific loggers.
type factory struct {
	root logr.Logger
}

func (f *factory) ContextLogger(
	provides *miruken.Provides,
	ctx miruken.HandleContext,
) (logr.Logger, miruken.HandleResult) {
	if parent := provides.Parent(); parent != nil {
		if pt, ok := parent.Key().(reflect.Type); ok {
			return f.root.WithName(pt.String()), miruken.Handled
		}
	}
	return f.root, miruken.Handled
}
