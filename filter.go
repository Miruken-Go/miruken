package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
)

// Filter stage priorities.
const (
	FilterStage              = 0
	FilterStageLogging       = 10
	FilterStageAuthorization = 30
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
			// self provided to facilitate late bindings
			self     Filter,
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

	// FilterAdapter is an adapter for implementing a
	// Filter using late binding method resolution.
	FilterAdapter struct {}

	// Filtered is a container of Filters.
	Filtered interface {
		Filters() []FilterProvider
		AddFilters(providers ...FilterProvider)
		RemoveFilters(providers ...FilterProvider)
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

func (n Next) PipeComposerAwait(
	composer Handler,
) []any {
	return mergeOutputAwait(n(composer, true))
}

func (n Next) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) ([]any, *promise.Promise[[]any], error) {
	var cb Callback
	if c, ok := callback.(Callback); ok {
		cb = c
	} else {
		var builder HandlesBuilder
		cb = builder.WithCallback(callback).New()
	}
	if result := composer.Handle(cb, greedy, nil); result.IsError() {
		return nil, nil, result.Error()
	} else if !result.handled {
		return nil, nil, &NotHandledError{callback}
	} else {
		if r, pr := cb.Result(greedy); pr != nil {
			return nil, promise.Then(pr, func(data any) []any {
				return []any{data}
			}), nil
		} else {
			return []any{r}, nil, nil
		}
	}
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

	// filterSpecProvider Resolves a Filter specification.
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
	p := builder.WithKey(spec.typ).New()
	resolve, pr, err := p.Resolve(composer, false)
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
	required bool,
	filters  ...Filter,
) *FilterInstanceProvider {
	return &FilterInstanceProvider{filters, required}
}


func (f *FilteredScope) Filters() []FilterProvider {
	return f.providers
}

func (f *FilteredScope) AddFilters(providers ...FilterProvider) {
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

func (f *FilteredScope) RemoveFilters(providers ...FilterProvider) {
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
	SkipFilters Option[bool]
}

var (
	DisableFilters = Options(FilterOptions{SkipFilters: Set(true)})
	EnableFilters  = Options(FilterOptions{SkipFilters: Set(false)})
)

func UseFilters(filters ...Filter) Builder {
	return withFilters(false, filters...)
}

func withFilters(required bool, filters ...Filter) Builder {
	return ProvideFilters(&FilterInstanceProvider{filters, required})
}

func ProvideFilters(providers ...FilterProvider) Builder {
	return Options(FilterOptions{Providers: providers})
}


// providedFilter models a Filter and its FilterProvider.
type providedFilter struct {
	filter   Filter
	provider FilterProvider
}


func orderedFilters(
	handler   Handler,
	binding   Binding,
	callback  Callback,
	providers ...[]FilterProvider,
) ([]providedFilter, error) {
	options, _  := GetOptions[FilterOptions](handler)
	skipFilters := options.SkipFilters
	bindingSkip := binding.SkipFilters()
	var allProviders []FilterProvider
	var addProvider = func (p FilterProvider) {
		if p == nil {
			return
		}
		if skipFilters.Set() {
			if skipFilters.Value() && !p.Required() {
				return
			}
		} else if bindingSkip && !p.Required() {
			return
		}
		if ap, ok := p.(interface {
			AppliesTo(Callback) bool
		}); ok {
			if !ap.AppliesTo(callback) {
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
			addProvider(p)
		}
	}
	if ps := options.Providers; ps != nil {
		for _, p := range ps {
			addProvider(p)
		}
	}
	if skipFilters != Set(true) {
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


func pipeline(
	ctx      HandleContext,
	filters  []providedFilter,
	complete func(HandleContext) ([]any, *promise.Promise[[]any], error),
) (results []any, pr *promise.Promise[[]any], err error) {
	index, length := 0, len(filters)
	var next Next
	next = func(
		composer Handler,
		proceed  bool,
	) ([]any, *promise.Promise[[]any], error) {
		if !proceed {
			return nil, nil, &RejectedError{ctx.Callback}
		}
		if composer != nil {
			ctx.Composer = composer
		}
		if index < length {
			pf := filters[index]
			f  := pf.filter
			index++
			return f.Next(f, next, ctx, pf.provider)
		}
		return complete(ctx)
	}

	return next(ctx.Composer, true)
}


// filterBinding describes the method used by a
// FilterAdapter to execute the Filter dynamically.
type filterBinding struct {
	method reflect.Method
	args   []arg
	ctxIdx int
	prvIdx int
}

func (n filterBinding) invoke(
	filter   Filter,
	ctx      HandleContext,
	next     Next,
	provider FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	initArgs := []any{filter, next}
	for i := 2; i <= 3; i++ {
		if n.ctxIdx == i {
			initArgs = append(initArgs, ctx)
		} else if n.prvIdx == i {
			initArgs = append(initArgs, provider)
		}
	}
	return callFunc(n.method.Func, ctx, n.args, initArgs...)
}


func (l FilterAdapter) Next(
	self     Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	var binding filterBinding
	if binding, err = getNextLate(self); err != nil {
		return
	}
	if out, pout, err = binding.invoke(self, ctx, next, provider); err != nil {
		return
	} else if pout == nil {
		pout, _ = out[1].(*promise.Promise[[]any])
		err,  _ = out[2].(error)
		out,  _ = out[0].([]any)
	} else {
		pout = promise.Then(pout, func(o []any) []any {
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
}


// getNextLate discovers a suitable "NextLate" method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getNextLate(
	filter Filter,
) (filterBinding, error) {
	typ := reflect.TypeOf(filter)
	if bindings := filterBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return binding, nil
		}
	}
	filterBindingLock.Lock()
	defer filterBindingLock.Unlock()
	bindings := filterBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return binding, nil
		}
		fb := make(map[reflect.Type]filterBinding)
		for k, v := range *bindings {
			fb[k] = v
		}
		bindings = &fb
	} else {
		fb := make(map[reflect.Type]filterBinding)
		bindings = &fb
	}
	if lateNext, ok := typ.MethodByName("NextLate"); !ok {
		goto Invalid
	} else if lateNextType := lateNext.Type;
		lateNextType.NumIn() < 2 || lateNextType.NumOut() < 3 {
		goto Invalid
	} else if lateNextType.In(1) != nextFuncType ||
		lateNextType.Out(0) != anySliceType ||
		lateNextType.Out(1) != promiseAnySliceType ||
		lateNextType.Out(2) != errorType {
		goto Invalid
	} else {
		skip    := 2 // skip receiver
		numArgs := lateNextType.NumIn()
		binding := filterBinding{method: lateNext}
		for i := 2; i < 4 && i < numArgs; i++ {
			if lateNextType.In(i) == handleCtxType {
				if binding.ctxIdx > 0 {
					return filterBinding{}, &MethodBindingError{lateNext,
						fmt.Errorf("filter: %v has duplicate HandleContext arg at index %v and %v",
							typ, binding.ctxIdx, i)}
				}
				binding.ctxIdx = i
				skip++
			} else if lateNextType.In(i) == filterProviderType {
				if binding.prvIdx > 0 {
					return filterBinding{}, &MethodBindingError{lateNext,
						fmt.Errorf("filter: %v has duplicate FilterProvider arg at index %v and %v",
							typ, binding.prvIdx, i)}
				}
				binding.prvIdx = i
				skip++
			}
		}
		args := make([]arg, numArgs-skip)
		if err := buildDependencies(lateNextType, skip, numArgs, args, 0); err != nil {
			err = fmt.Errorf("filter: %v \"NextLate\": %w", typ, err)
			return filterBinding{}, &MethodBindingError{lateNext, err}
		}
		binding.args = args
		(*bindings)[typ] = binding
		filterBindingMap.Store(bindings)
		return binding, nil
	}
Invalid:
	return filterBinding{}, fmt.Errorf(`filter: %v has no valid "NextLate" method`, typ)
}


var (
	filterBindingLock sync.Mutex
	nextFuncType        = internal.TypeOf[Next]()
	filterProviderType  = internal.TypeOf[FilterProvider]()
	filterBindingMap    = atomic.Pointer[map[reflect.Type]filterBinding]{}
	promiseAnySliceType = internal.TypeOf[*promise.Promise[[]any]]()
)
