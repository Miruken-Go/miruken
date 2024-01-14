package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	context2 "github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Options is used to customize the setup process.
	Options struct {
		StartupTimeout  time.Duration
		ShutdownTimeout time.Duration
	}

	// Bootstrap is used to customize application startup and shutdown.
	// All Bootstrap instances are resolved during New setup and Startup
	// invoked concurrently.  Shutdown is invoked in reverse order.
	Bootstrap interface {
		Startup(
			ctx context.Context,
			h   miruken.Handler,
		) *promise.Promise[struct{}]

		Shutdown(
			ctx context.Context,
		) *promise.Promise[struct{}]
	}

	bootstrapper struct {
		options    Options
		bootstraps []Bootstrap
	}
)


func (b *bootstrapper) Constructor(
	_ *struct {
		provides.It
		context2.Scoped
	  },
	_ *struct {
		args.Optional
		args.FromOptions
	  }, options Options,
	bootstraps []Bootstrap,
) {
	b.options    = options
	b.bootstraps = bootstraps
}

func (b *bootstrapper) bootstrap(
	h miruken.Handler,
) *promise.Promise[struct{}] {
	if bootstraps := b.bootstraps; len(bootstraps) > 0 {
		ctx := context.Background()
		var cancel context.CancelFunc
		if timeout := b.options.StartupTimeout; timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
		}
		promises := make([]*promise.Promise[struct{}], len(bootstraps))
		for i, bootstrap := range bootstraps {
			promises[i] = bootstrap.Startup(ctx, h)
		}
		return promise.Erase(promise.All(ctx, promises...)).OnCancel(cancel)
	}
	return promise.Empty()
}


func (b *bootstrapper) Dispose() {
	if bootstraps := b.bootstraps; len(bootstraps) > 0 {
		ctx := context.Background()
		if timeout := b.options.ShutdownTimeout; timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		promises := make([]*promise.Promise[struct{}], len(bootstraps))
		for i := range bootstraps {
			bootstrap := bootstraps[len(bootstraps)-1-i]
			promises[i] = bootstrap.Shutdown(ctx)
		}
		if _, err := promise.All(ctx, promises...).Await(); err != nil {
			panic(fmt.Errorf("failed to gracefully shutdown: %w", err))
		}
	}
}