package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sort"
	"sync"
)

// Filter stage priorities.
const (
	FilterStage              = 0
	FilterStageLogging       = 10
	FilterStateAuthorization = 30
	FilterStageValidation    = 50
)

type (
	// Next advances to the next step in a pipeline.
	Next func (
		composer Handler,
		proceed  bool,
	) ([]any, *promise.Promise[[]any], error)

	// Filter defines a middleware step in a pipeline.
	Filter interface {
		Order() int
		Next(
			next     Next,
			ctx      HandleContext,
			provider FilterProvider,
		) ([]any, *promise.Promise[[]any], error)
	}

	// FilterProvider provides one or more Filter's.
	FilterProvider interface {
		Required() bool
		Filters(
			binding  Binding,
			callback any,
			composer Handler,
		) ([]Filter, error)
	}

	// FilterInstanceProvider provides existing Filters.
	FilterInstanceProvider struct {
		filters  []Filter
		required bool
	}

	// Filtered is a container of Filters.
	Filtered interface {
		Filters() []FilterProvider
		AddFilters(providers ... FilterProvider)
		RemoveFilters(providers ... FilterProvider)
		RemoveAllFilters()
	}

	// FilteredScope implements a container of Filters.
	FilteredScope struct {
		providers []FilterProvider
	}
)

func (n Next) Pipe() ([]any, *promise.Promise[[]any], error) {
	return mergeOutput(n(nil, true))
}

func (n Next) PipeAwait() []any {
	return mergeOutputAwait(n(nil, true))
}

func (n Next) PipeComposer(
	composer Handler,
) ([]any, *promise.Promise[[]any], error) {
	return mergeOutput(n(composer, true))
}

func (n Next) PipeComposerAsync(
	composer Handler,
) []any {
	return mergeOutputAwait(n(composer, true))
}

func (n Next) Abort() ([]any, *promise.Promise[[]any], error) {
	return n(nil, false)
}

func (n Next) Fail(err error) ([]any, *promise.Promise[[]any], error) {
	return nil, nil, err
}

type (
	// filterSpec describes a Filter.
	filterSpec struct {
		typ      reflect.Type
		required bool
		order    int
	}

	// filterSpecProvider resolves a Filter specification.
	filterSpecProvider struct {
		spec filterSpec
	}
)

func (f *filterSpecProvider) Required() bool {
	return f.spec.required
}

func (f *filterSpecProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	spec := f.spec
	var builder ProvidesBuilder
	provides := builder.WithKey(spec.typ).NewProvides()
	resolve, pr, err := provides.Resolve(composer, false)
	if err != nil {
		return nil, err
	}
	if pr != nil {
		resolve, err = pr.Await()
	}
	if resolve != nil && err == nil {
		if filter, ok := resolve.(Filter); ok {
			if spec.order >= 0 {
				if o, ok := filter.(interface{ SetOrder(order int) }); ok {
					o.SetOrder(spec.order)
				}
			}
			return []Filter{filter}, nil
		}
	}
	return nil, err
}

func (f *FilterInstanceProvider) Required() bool {
	return f.required
}

func (f *FilterInstanceProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return f.filters, nil
}

func NewFilterInstanceProvider(
	required    bool,
	filters ... Filter,
) *FilterInstanceProvider {
	return &FilterInstanceProvider{filters, required}
}

func (f *FilteredScope) Filters() []FilterProvider {
	return f.providers
}

func (f *FilteredScope) AddFilters(providers ... FilterProvider) {
	if len(providers) == 0 {
		return
	}
	Loop:
	for _, fp := range providers {
		if fp == nil {
			panic("provider cannot be nil")
		}
		for _, sfp := range f.providers {
			if sfp == fp {
				continue Loop
			}
		}
		f.providers = append(f.providers, fp)
	}
}

func (f *FilteredScope) RemoveFilters(providers ... FilterProvider) {
	if len(providers) == 0 {
		return
	}
	for _, fp := range providers {
		if fp == nil {
			panic("provider cannot be nil")
		}
		for i, sfp := range f.providers {
			if sfp == fp {
				f.providers = append(f.providers[:i], f.providers[i+1:]...)
				break
			}
		}
	}
}

func (f *FilteredScope) RemoveAllFilters() {
	f.providers = nil
}

// Filter builders

// FilterOptions are used to control Filter processing.
type FilterOptions struct {
	Providers   []FilterProvider
	SkipFilters OptionBool
}

var (
	disableFilters = Options(FilterOptions{
		SkipFilters: OptionTrue,
	})

	enableFilters = Options(FilterOptions{
		SkipFilters: OptionFalse,
	})

	DisableFilters BuilderFunc = func (handler Handler) Handler {
		return BuildUp(handler, disableFilters)
	}

	EnableFilters BuilderFunc = func (handler Handler) Handler {
		return BuildUp(handler, enableFilters)
	}
)

func UseFilters(filters ... Filter) Builder {
	return withFilters(false, filters...)
}

func RequireFilters(filters ... Filter) Builder {
	return withFilters(true, filters...)
}

func withFilters(required bool, filters ... Filter) Builder {
	provider := FilterInstanceProvider{filters, required}
	builder  := Options(FilterOptions{
		Providers: []FilterProvider{&provider},
	})
	return BuilderFunc(func (handler Handler) Handler {
		return BuildUp(handler, builder)
	})
}

func ProvideFilters(providers ... FilterProvider) BuilderFunc {
	builder := Options(FilterOptions{Providers: providers})
	return func (handler Handler) Handler {
		return BuildUp(handler, builder)
	}
}

// providedFilter models a Filter and its FilterProvider.
type providedFilter struct {
	filter   Filter
	provider FilterProvider
}

func orderedFilters(
	handler       Handler,
	binding       Binding,
	callback      Callback,
	providers ... []FilterProvider,
) ([]providedFilter, error) {
	var options FilterOptions
	GetOptions(handler, &options)
	skipFilters := options.SkipFilters
	bindingSkip := binding.SkipFilters()
	var allProviders []FilterProvider
	var addProvider = func (p FilterProvider) {
		switch skipFilters {
		case OptionTrue:
			if !p.Required() {
				return
			}
		case OptionNone:
			if bindingSkip && !p.Required() {
				return
			}
		}
		allProviders = append(allProviders, p)
	}
	for _, ps := range providers {
		if ps == nil {
			continue
		}
		for _, p := range ps {
			if p == nil {
				continue
			}
			if ap, ok := p.(interface {
				AppliesTo(Callback) bool
			}); ok {
				if !ap.AppliesTo(callback) {
					continue
				}
			}
			addProvider(p)
		}
	}
	for _, p := range options.Providers {
		if p != nil {
			addProvider(p)
		}
	}
	if skipFilters != OptionTrue {
		handler = BuildUp(handler, DisableFilters)
	}
	var allFilters []providedFilter
	for _, provider := range allProviders {
		found := false
		filters, err := provider.Filters(binding, callback, handler)
		if filters == nil || err != nil {
			return nil, err
		}
		for _, filter := range filters {
			if filter == nil {
				return nil, nil
			}
			found = true
			allFilters = append(allFilters, providedFilter{
				filter, provider,
			})
		}
		if !found {
			return nil, nil
		}
	}
	if allFilters == nil {
		return []providedFilter{}, nil
	}
	sort.Slice(allFilters, func(i, j int) bool {
		filter1, filter2 := allFilters[i].filter, allFilters[j].filter
		if filter1 == filter2 {
			return false
		}
		order1, order2 := filter1.Order(), filter2.Order()
		if order1 == order2 || order2 < 0 {
			return true
		}
		if order1 < 0 {
			return false
		}
		return order1 < order2
	})
	return allFilters, nil
}

type CompletePipelineFunc func(
	HandleContext,
) ([]any, *promise.Promise[[]any], error)

func pipeline(
	ctx      HandleContext,
	filters  []providedFilter,
	complete CompletePipelineFunc,
) (results []any, pr *promise.Promise[[]any], err error) {
	index, length := 0, len(filters)
	var next Next
	next = func(
		composer Handler,
		proceed  bool,
	) ([]any, *promise.Promise[[]any], error) {
		if !proceed {
			return nil, nil, RejectedError{ctx.Callback()}
		}
		if composer != nil {
			ctx.composer = composer
		}
		if index < length {
			pf := filters[index]
			index++
			return pf.filter.Next(next, ctx, pf.provider)
		}
		return complete(ctx)
	}

	return next(ctx.Composer(), true)
}

func DynNext(
	filter   Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	typ := reflect.TypeOf(filter)
	_dynNextLock.RLock()
	binding := _dynNextBinding[typ]
	_dynNextLock.RUnlock()
	if binding == nil {
		_dynNextLock.Lock()
		defer _dynNextLock.Unlock()
		if dynNext, ok := typ.MethodByName("DynNext"); !ok {
			goto Invalid
		} else if dynNextType := dynNext.Type;
				dynNextType.NumIn() < 4 || dynNextType.NumOut() < 3 {
			goto Invalid
		} else if dynNextType.In(1) != reflect.TypeOf(next) ||
			dynNextType.In(2) != _handleCtxType ||
			dynNextType.In(3) != _filterProviderType ||
			dynNextType.Out(0) != _anySliceType ||
			dynNextType.Out(1) != _promiseAnySliceType ||
			dynNextType.Out(2) != _errorType {
			goto Invalid
		} else {
			numArgs := dynNextType.NumIn()
			args    := make([]arg, numArgs-4)  // skip receiver
			if inv := buildDependencies(dynNextType, 4, numArgs, args, 0); inv != nil {
				err = fmt.Errorf("DynNext: %w", inv)
			}
			if err != nil {
				return nil, nil, MethodBindingError{dynNext, err}
			}
			binding = &nextBinding{dynNext, args}
			_dynNextBinding[typ] = binding
		}
	}
	if out, po, err = binding.invoke(ctx, filter, next, ctx, provider); err != nil {
		return
	} else if po == nil {
		po,  _ = out[1].(*promise.Promise[[]any])
		err, _ = out[2].(error)
		out, _ = out[0].([]any)
	} else {
		po = promise.Then(po, func(o []any) []any {
			if err, ok := o[2].(error); ok {
				panic(err)
			} else if ro, ok := o[0].([]any); ok {
				return ro
			} else {
				return nil
			}
		})
	}
	return

	Invalid:
		return nil, nil, fmt.Errorf(
			"filter %v requires a method DynNext(Next, HandleContext, FilterProvider, ...)",
			typ)
}

type nextBinding struct {
	method reflect.Method
	args   []arg
}

func (n *nextBinding) invoke(
	ctx      HandleContext,
	initArgs ... any,
) ([]any, *promise.Promise[[]any], error) {
	return callFunc(n.method.Func, ctx, n.args, initArgs...)
}

var (
	_dynNextLock sync.RWMutex
	_dynNextBinding = make(map[reflect.Type]*nextBinding)
)
