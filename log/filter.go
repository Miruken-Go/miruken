package log

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"time"
)

type (
	// Provider is a FilterProvider for logging.
	Provider struct {
		verbosity int
	}

	// filter logs basic callback execution details.
	filter struct {}
)

const durationFormat = "15:04:05.000000"  // microseconds

// Provider

func (l *Provider) InitWithTag(tag reflect.StructTag) error {
	if log, ok := tag.Lookup("log"); ok {
		_, err := fmt.Sscanf(log, "verbosity=%d", &l.verbosity)
		return err
	}
	return nil
}

func (l *Provider) Required() bool {
	return false
}

func (l *Provider) AppliesTo(
	callback miruken.Callback,
) bool {
	_, ok := callback.(*miruken.Handles)
	return ok
}

func (l *Provider) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return filters, nil
}

// filter

func (f filter) Order() int {
	return miruken.FilterStageLogging
}

func (f filter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if lp, ok := provider.(*Provider); ok {
		logger, _, re := miruken.Resolve[logr.Logger](ctx.Composer())
		if re != nil {
			return next.Pipe()
		}
		if logger = logger.V(lp.verbosity); !logger.Enabled() {
			return next.Pipe()
		}
		logger = logger.WithName(fmt.Sprintf("%T", ctx.Handler()))
		callback := ctx.Callback()
		logger.Info("handling",
			"callback", reflect.TypeOf(callback).String(),
			"source", callback.Source())
		start := time.Now()
		if out, pout, err = next.Pipe(); err != nil {
			f.logError(err, start, logger)
			return
		} else if pout == nil {
			f.logSuccess(start, logger)
			return
		} else {
			return nil, promise.Catch(
				promise.Then(pout, func(oo []any) []any {
					f.logSuccess(start, logger)
					return oo
				}), func(ee error) error {
					f.logError(ee, start, logger)
					return ee
				}), nil
		}
	}
	return next.Abort()
}

func (f filter) logSuccess(
	start  time.Time,
	logger logr.Logger,
) {
	elapsed := miruken.Timespan(time.Since(start))
	logger.Info("completed", "duration", elapsed.Format(durationFormat))
}

func (f filter) logError(
	err    error,
	start  time.Time,
	logger logr.Logger,
) {
	elapsed := miruken.Timespan(time.Since(start))
	logger.Error(err, "failed", "duration", elapsed.Format(durationFormat))
}

var filters = []miruken.Filter{filter{}}
