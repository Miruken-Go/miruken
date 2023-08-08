package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"math"
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
	// The optional composer replaces the Handler in the next step.
	// The optional values provide dependencies to the next step.
	Next func (
		composer Handler,
		proceed  bool,
		values   ...any,
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

	// FilterAdapter is an adapter for implementing a
	// Filter using late binding method resolution.
	FilterAdapter struct {}

	// FilterInstanceProvider provides existing Filters.
	FilterInstanceProvider struct {
		filters  []Filter
		required bool
	}

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


// Next

func (n Next) Pipe(values ...any,) ([]any, *promise.Promise[[]any], error) {
	return mergeOutput(n(nil, true, values...))
}

func (n Next) PipeAwait(values ...any,) []any {
	return mergeOutputAwait(n(nil, true, values...))
}

func (n Next) PipeComposer(
	composer Handler,
	values   ...any,
) ([]any, *promise.Promise[[]any], error) {
	return mergeOutput(n(composer, true, values...))
}

func (n Next) PipeComposerAwait(
	composer Handler,
	values   ...any,
) []any {
	return mergeOutputAwait(n(composer, true, values...))
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


// FilterAdapter

func (l FilterAdapter) Next(
	self     Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	if group, err := getFilterBinding(self); err == nil {
		return group.invoke(self, ctx, next, provider)
	} else {
		return nil, nil, err
	}
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


// filterSpecProvider

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


// FilterInstanceProvider

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


// FilteredScope

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

type (
	// FilterOptions are used to control Filter processing.
	FilterOptions struct {
		Providers   []FilterProvider
		SkipFilters Option[bool]
	}
)

var (
	DisableFilters = Options(FilterOptions{SkipFilters: Set(true)})
	EnableFilters  = Options(FilterOptions{SkipFilters: Set(false)})
)

func UseFilters(filters ...Filter) Builder {
	return ProvideFilters(&FilterInstanceProvider{filters, false})
}

func ProvideFilters(providers ...FilterProvider) Builder {
	return Options(FilterOptions{Providers: providers})
}


type (
	// providedFilter models a Filter and its FilterProvider.
	providedFilter struct {
		filter   Filter
		provider FilterProvider
	}
)


func orderFilters(
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
		values   ...any,
	) ([]any, *promise.Promise[[]any], error) {
		if !proceed {
			return nil, nil, &RejectedError{ctx.Callback}
		}
		if composer != nil {
			ctx.Composer = composer
		}
		if len(values) > 0 {
			ctx.Composer = BuildUp(ctx.Composer, With(values...))
		}
		if index < length {
			pf := filters[index]
			f  := pf.filter
			index++
			return f.Next(f, next, ctx, pf.provider)
		}
		return complete(ctx)
	}

	return next(nil, true)
}


type (
	// filterBinding executes a Filter method dynamically.
	filterBinding struct {
		method reflect.Method
		args   []arg
		ctxIdx int
		prvIdx int
	}

	// filterBindingGroup executes a chain of Filter's dynamically.
	filterBindingGroup []filterBinding

	// compoundHandler uses dynamic Filter's to split callback handling
	// into pure and impure steps for better clarity and testability
	compoundHandler struct {
		handler any
		filters filterBindingGroup
	}
)


func (n filterBinding) invoke(
	filter   any,
	ctx      HandleContext,
	next     Next,
	provider FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	initArgs := []any{filter, next}
	for i := 2; i <= 3; i++ {
		if n.ctxIdx == i {
			initArgs = append(initArgs, ctx)
		} else if n.prvIdx == i {
			initArgs = append(initArgs, provider)
		}
	}
	if out, pout, err = callFunc(n.method.Func, ctx, n.args, initArgs...); err != nil {
		return
	} else if pout == nil {
		pout, _ = out[1].(*promise.Promise[[]any])
		err,  _ = out[2].(error)
		out,  _ = out[0].([]any)
		return
	} else {
		pout = promise.Then(pout, func(o []any) []any {
			if err, ok := o[2].(error); ok {
				panic(err)
			} else if ro, ok := o[0].([]any); ok {
				return ro
			}
			return nil
		})
	}
	return
}


func (g filterBindingGroup) invoke(
	filter   any,
	ctx      HandleContext,
	next     Next,
	provider FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	if len(g) == 1 {
		return g[0].invoke(filter, ctx, next, provider)
	}
	index, length := 0, len(g)
	var n Next
	n = func(
		composer Handler,
		proceed  bool,
		values   ...any,
	) ([]any, *promise.Promise[[]any], error) {
		if !proceed {
			return nil, nil, &RejectedError{ctx.Callback}
		}
		if composer != nil {
			ctx.Composer = composer
		}
		if index < length {
			fb := g[index]
			index++
			if len(values) > 0 {
				ctx.Composer = BuildUp(ctx.Composer, With(values...))
			}
			return fb.invoke(filter, ctx, n, provider)
		}
		return next(ctx.Composer, true, values...)
	}
	return n(nil, true)
}


func (c compoundHandler) Order() int {
	return math.MaxInt32
}

func (c compoundHandler) Next(
	self     Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) ([]any, *promise.Promise[[]any], error) {
	if filters := c.filters; filters != nil {
		return filters.invoke(c.handler, ctx, next, provider)
	} else {
		return next(nil, true)
	}
}


// getFilterBinding discovers a suitable dynamic Filter binding.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
// If ignoreNext is true, the "Next" Filter method will be ignored.
func getFilterBinding(
	filter Filter,
) (filterBindingGroup, error) {
	typ := reflect.TypeOf(filter)
	if bindings := filterBindingMap.Load(); bindings != nil {
		if group, ok := (*bindings)[typ]; ok {
			return group, nil
		}
	}
	filterBindingLock.Lock()
	defer filterBindingLock.Unlock()
	bindings := filterBindingMap.Load()
	if bindings != nil {
		if group, ok := (*bindings)[typ]; ok {
			return group, nil
		}
		fb := make(map[reflect.Type]filterBindingGroup, len(*bindings)+1)
		for k, v := range *bindings {
			fb[k] = v
		}
		bindings = &fb
	} else {
		bindings = &map[reflect.Type]filterBindingGroup{}
	}
	var group filterBindingGroup
	// Methods in GO are sorted in lexicographic order which will
	// determine the order of filter execution.
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name != "Next" {
			if binding, err := parseFilterMethod(method); err != nil {
				return nil, err
			} else if binding != nil {
				group = append(group, *binding)
			}
		}
	}
	if len(group) > 0 {
		(*bindings)[typ] = group
		filterBindingMap.Store(bindings)
		return group, nil
	}
	return nil, fmt.Errorf(`filter: %v has no compatible dynamic methods`, typ)
}

// parseFilterMethod parses a method to see if it is a suitable dynamic Filter method.
func parseFilterMethod(
	method reflect.Method,
) (*filterBinding, error) {
	if funcType := method.Type;
		funcType.NumIn() < 2 || funcType.NumOut() < 3 {
		return nil, nil
	} else if funcType.In(1) != nextFuncType ||
		funcType.Out(0) != anySliceType ||
		funcType.Out(1) != promiseAnySliceType ||
		funcType.Out(2) != errorType {
		return nil, nil
	} else {
		skip    := 2 // skip receiver
		numArgs := funcType.NumIn()
		binding := filterBinding{method: method}
		for i := 2; i < 4 && i < numArgs; i++ {
			if funcType.In(i) == handleCtxType {
				if binding.ctxIdx > 0 {
					return nil, &MethodBindingError{method,
						fmt.Errorf(
							"filter: %v %q has duplicate HandleContext arg at index %v and %v",
							funcType.In(0), method.Name, binding.ctxIdx, i)}
				}
				binding.ctxIdx = i
				skip++
			} else if funcType.In(i) == filterProviderType {
				if binding.prvIdx > 0 {
					return nil, &MethodBindingError{method,
						fmt.Errorf(
							"filter: %v %q has duplicate FilterProvider arg at index %v and %v",
							funcType.In(0), method.Name, binding.prvIdx, i)}
				}
				binding.prvIdx = i
				skip++
			}
		}
		args := make([]arg, numArgs-skip)
		if err := buildDependencies(funcType, skip, numArgs, args, 0); err != nil {
			err = fmt.Errorf("filter: %v %q: %w", funcType.In(0), method.Name, err)
			return nil, &MethodBindingError{method, err}
		}
		binding.args = args
		return &binding, nil
	}
}


var (
	filterBindingLock sync.Mutex
	nextFuncType        = internal.TypeOf[Next]()
	filterProviderType  = internal.TypeOf[FilterProvider]()
	filterBindingMap    = atomic.Pointer[map[reflect.Type]filterBindingGroup]{}
	promiseAnySliceType = internal.TypeOf[*promise.Promise[[]any]]()
)
