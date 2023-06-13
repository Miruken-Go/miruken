package logs

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken/provides"
)

// Factory of context specific loggers.
type Factory struct {
	root logr.Logger
}

// NewContextLogger return a new logger in a context.
// The context is a name derived from the following information.
// If the request has an owner, the owner's type is used.
// Otherwise, the root logger is returned.
func (f *Factory) NewContextLogger(
	p *provides.It,
) logr.Logger {
	if owner := p.Owner(); owner != nil {
		return f.root.WithName(fmt.Sprintf("%T", owner))
	}
	return f.root
}
