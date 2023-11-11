package logs

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"reflect"
	"time"
)

type (
	// Emit is a FilterProvider for logging.
	Emit struct {
		verbosity int
	}

	// filter logs basic callback execution details.
	filter struct {}
)

const durationFormat = "15:04:05.000000"  // microseconds

// Emit

func (e *Emit) InitWithTag(tag reflect.StructTag) error {
	if log, ok := tag.Lookup("logs"); ok {
		_, err := fmt.Sscanf(log, "verbosity=%d", &e.verbosity)
		return err
	}
	return nil
}

func (e *Emit) Required() bool {
	return false
}

func (e *Emit) AppliesTo(
	callback miruken.Callback,
) bool {
	_, ok := callback.(*handles.It)
	return ok
}

func (e *Emit) Filters(
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
	self     miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if emit, ok := provider.(*Emit); ok {
		logger, _, ok, re := provides.Type[logr.Logger](ctx)
		if !(ok && re == nil) {
			return next.Pipe()
		}
		if logger = logger.V(emit.verbosity); !logger.Enabled() {
			return next.Pipe()
		}
		logger = logger.WithName(fmt.Sprintf("%T", ctx.Handler))
		callback := ctx.Callback
		source   := callback.Source()
		logger.Info("handling",
			"callback", reflect.TypeOf(callback),
			"source-type", reflect.TypeOf(source),
			"source",source)
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
