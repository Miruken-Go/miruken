package miruken

import (
	"reflect"
	"sort"
)

type Next func (
	composer Handler,
	proceed  bool,
) ([]interface{}, error)

func (n Next) Filter() ([]interface{}, error) {
	return n(nil, true)
}

func (n Next) WithComposer(composer Handler) ([]interface{}, error) {
	return n(composer, true)
}

func (n Next) Abort() ([]interface{}, error) {
	return n(nil, false)
}

// Filter defines the smallest step in a pipeline.
type Filter interface {
	Order() int
	Next(
		next     Next,
		context  HandleContext,
		provider FilterProvider,
	)  ([]interface{}, error)
}

// FilterProvider provides one or more Filter's.
type FilterProvider interface {
	Required() bool
	Filters(
		binding  Binding,
		callback interface{},
		composer Handler,
	) ([]Filter, error)
}

// filterSpec describes a Filter.
type filterSpec struct {
	typ      reflect.Type
	required bool
	order    int
}

// filterSpecProvider resolves a Filter specification.
type filterSpecProvider struct {
	spec filterSpec
}

func (f *filterSpecProvider) Required() bool {
	return f.spec.required
}

func (f *filterSpecProvider) Filters(
	binding  Binding,
	callback interface{},
	composer Handler,
) ([]Filter, error) {
	spec    := f.spec
	inquiry := new(InquiryBuilder).
		WithKey(spec.typ).
		NewInquiry()
	result, err := inquiry.Resolve(composer)
	if result != nil && err != nil {
		if filter, ok := result.(Filter); ok {
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

// filterInstanceProvider manages existing instances.
type filterInstanceProvider struct {
	required bool
	filters  []Filter
}

func (f *filterInstanceProvider) Required() bool {
	return f.required
}

func (f *filterInstanceProvider) Filters(
	binding  Binding,
	callback interface{},
	composer Handler,
) ([]Filter, error) {
	return f.filters, nil
}

// Filtered represents a container of Filter's.
type Filtered interface {
	Filters() []FilterProvider
	AddFilters(providers ... FilterProvider)
	RemoveFilters(providers ... FilterProvider)
	RemoveAllFilters()
}

// FilteredScope is a container for Filters.
type FilteredScope struct {
	providers []FilterProvider
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

// FilterOptions are used the control filter processing.
type FilterOptions struct {
	SkipFilters OptionBool
	Providers   []FilterProvider
}

var skipFilters = WithOptions(FilterOptions{
	SkipFilters: OptionTrue,
})
var SkipFilters = BuilderFunc(func (handler Handler) Handler {
	return Build(handler, skipFilters)
})

var enableFilters = WithOptions(FilterOptions{
	SkipFilters: OptionFalse,
})
var EnableFilters = BuilderFunc(func (handler Handler) Handler {
	return Build(handler, enableFilters)
})

func WithFilters(filters ... Filter) Builder {
	return withFilters(false, filters...)
}

func WithRequiredFilters(filters ... Filter) Builder {
	return withFilters(true, filters...)
}

func withFilters(required bool, filters ... Filter) Builder {
	provider := filterInstanceProvider{required, filters}
	builder  := WithOptions(FilterOptions{
		Providers: []FilterProvider{&provider},
	})
	return BuilderFunc(func (handler Handler) Handler {
		return Build(handler, builder)
	})
}

func WithFilterProviders(providers ... FilterProvider) Builder {
	builder := WithOptions(FilterOptions{
		Providers: providers,
	})
	return BuilderFunc(func (handler Handler) Handler {
		return Build(handler, builder)
	})
}

// providedFilter represents a Filter and its FilterProvider.
type providedFilter struct {
	filter   Filter
	provider FilterProvider
}

func orderedFilters(
	handler       Handler,
	binding       Binding,
	callback      interface{},
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
				AppliesTo(callback interface{}) bool }); ok {
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
		handler = Build(handler, SkipFilters)
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

type CompletePipelineFunc func(HandleContext) ([]interface{}, error)

func pipeline(
	context  HandleContext,
	filters  []providedFilter,
	complete CompletePipelineFunc,
) (results []interface{}, err error) {
	callback := context.Callback
	composer := context.Composer
	index, length := 0, len(filters)

	var next Next
	next = func(
		comp     Handler,
		proceed  bool,
	) ([]interface{}, error) {
		if !proceed {
			return nil, RejectedError{callback}
		}
		if comp != nil {
			composer = comp
		}
		if index < length {
			pf := filters[index]
			index++
			return pf.filter.Next(next, context, pf.provider)
		}
		return complete(context)
	}

	return next(composer, true)
}