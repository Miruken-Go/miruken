package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
)

type (
	// Lifestyle provides common lifestyle functionality.
	Lifestyle struct{}

	// LifestyleProvider is a FilterProvider of lifestyles.
	LifestyleProvider struct {
		filters []Filter
	}

	// LifestyleInit initializes a Lifestyle from Binding info.
	LifestyleInit interface {
		InitLifestyle(Binding) error
	}
)


// Lifestyle

func (l *Lifestyle) Order() int {
	return math.MaxInt32 - 1000
}


// LifestyleProvider

func (l *LifestyleProvider) Required() bool {
	return true
}

func (l *LifestyleProvider) AppliesTo(
	callback Callback,
) bool {
	_, ok := callback.(*Provides)
	return ok
}

func (l *LifestyleProvider) Filters(
	binding Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return l.filters, nil
}

func (l *LifestyleProvider) FiltersAssigned() bool {
	return l.filters != nil
}

func (l *LifestyleProvider) SetFilters(filters ...Filter) {
	l.filters = filters
}

func (l *LifestyleProvider) Constraints() []Constraint {
	return enforceLifestyle
}


// Single

type (
	// Single LifestyleProvider providing same instance.
	Single struct {
		LifestyleProvider
	}

	// singleEntry stores a lazy instance.
	singleEntry struct {
		instance []any
		once     *sync.Once
	}

	// singleCache maintains a cache of singleEntry's.
	singleCache map[any]*singleEntry

	// single is a Filter that caches a known instance.
	single struct {
		Lifestyle
		entry singleEntry
	}

	// singleUnk is a Filter that caches unknown instances.
	// When a Handler provides any results, a map of key to
	// instance is maintained using copy-on-write idiom.
	singleUnk struct {
		Lifestyle
		keys atomic.Pointer[singleCache]
		lock sync.Mutex
	}
)


// Single

func (s *Single)InitLifestyle(binding Binding) error {
	if !s.FiltersAssigned(){
		if typ, ok := binding.Key().(reflect.Type); ok && internal.IsAny(typ) {
			s.SetFilters(&singleUnk{})
		} else {
			s.SetFilters(&single{entry: singleEntry{once: new(sync.Once)}})
		}
	}
	return nil
}


// single

func (s *single) Next(
	self     Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) (out []any, po *promise.Promise[[]any], err error) {
	return s.entry.get(next)
}


// singleUnk

func (s *singleUnk) Next(
	self     Filter,
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
) (out []any, po *promise.Promise[[]any], err error) {
	key := ctx.Callback.(*Provides).Key()

	var entry *singleEntry
	if keys := s.keys.Load(); keys != nil {
		if e, ok := (*keys)[key]; ok {
			entry = e
		}
	}

	// Use copy-on-write idiom since reads should be more frequent than writes.
	if entry == nil {
		s.lock.Lock()
		if keys := s.keys.Load(); keys != nil {
			if entry = (*keys)[key]; entry == nil {
				kc := make(singleCache, len(*keys)+1)
				typ, assignable := key.(reflect.Type)
				// If the key is not found, check if any existing instances
				// can satisfy the key before a new instance is provided.
				for k, v := range *keys {
					kc[k] = v
					if assignable {
						if instance := v.instance; len(instance) > 0 {
							if o := instance[0]; o != nil {
								if ot := reflect.TypeOf(o); ot.AssignableTo(typ) {
									entry   = v
									kc[key] = v
									break
								}
							}
						}
					}
				}
				if entry == nil {
					entry = &singleEntry{once: new(sync.Once)}
					kc[key] = entry
				}
				s.keys.Store(&kc)
			}
		} else {
			entry = &singleEntry{once: new(sync.Once)}
			s.keys.Store(&singleCache{key: entry})
		}
		s.lock.Unlock()
	}

	return entry.get(next)
}


// singleEntry

func (s *singleEntry) get(
	next Next,
) (out []any, po *promise.Promise[[]any], err error) {
	s.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("single: panic: %v", r)
				}
				s.once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			s.once = new(sync.Once)
		} else {
			s.instance = out
		}
	})
	return s.instance, nil, err
}


var (
	// UseLifestyle forces resolution from a handler with lifestyle.
	// This is used to suppress implied resolution values from context.
	UseLifestyle Qualifier[Lifestyle]

	// enforceLifestyle enforces lifestyle resolution.
	enforceLifestyle = []Constraint{UseLifestyle}
)