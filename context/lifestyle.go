package context

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"reflect"
	"sync"
)

type (
	// Lifestyle is a BindingGroup for requesting a scoped lifestyle.
	// Provides associates both the scoped lifestyle and fromScope constraint.
	Lifestyle struct {
		miruken.BindingGroup
		scoped
		miruken.Qualifier[fromScope]
	}

	// Rooted is a BindingGroup for requesting a rooted scoped lifestyle
	// with all resolutions assigned to the root Context.
	Rooted struct {
		miruken.BindingGroup
		scoped `scoped:"rooted"`
		miruken.Qualifier[fromScope]
	}

	// scoped LifestyleProvider provides instances per Context.
	scoped struct {
		miruken.LifestyleProvider
		rooted bool
	}

	// fromScope is used to constrain Provides from Context.
	fromScope struct {}
)

func (s *scoped) InitWithTag(tag reflect.StructTag) error {
	if scoped, ok := tag.Lookup("scoped"); ok {
		s.rooted = scoped == "rooted"
	}
	return s.Init()
}

func (s *scoped) Init() error {
	s.SetFilters(&scopedFilter{})
	return nil
}

// scopedFilter is a Filter that caches an instance per Context.
type scopedFilter struct {
	miruken.Lifestyle
	cache  map[*Context]miruken.LifestyleCache
	lock   sync.RWMutex
}

func (s *scopedFilter) Next(
	next miruken.Next,
	ctx miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	key := ctx.Callback().(*provides.It).Key()
	if key == contextType {
		// can't resolve a context contextually
		return nil, nil,nil
	}

	rooted := false
	if scp, ok := provider.(*scoped); ok {
		rooted = scp.rooted
	}

	if !s.isCompatibleWithParent(ctx, rooted) {
		return nil, nil,nil
	}
	context, _, err := miruken.Resolve[*Context](ctx.Composer())
	if err != nil {
		return nil, nil, err
	} else if context == nil {
		return next.Abort()
	} else if context.State() != StateActive {
		return nil, nil, errors.New("scoped: cannot scope instances to an inactive context")
	} else if rooted {
		context = context.Root()
	}

	var entry *miruken.LifestyleEntry
	s.lock.RLock()
	if cache := s.cache; cache != nil {
		if keys := cache[context]; keys != nil {
			entry = keys[key]
		}
	}
	s.lock.RUnlock()

	if entry == nil {
		s.lock.Lock()
		if cache := s.cache; cache != nil {
			if keys := cache[context]; keys != nil {
				if entry = keys[key]; entry == nil {
					entry     = &miruken.LifestyleEntry{Once: new(sync.Once)}
					keys[key] = entry
				}
			} else {
				entry = &miruken.LifestyleEntry{Once: new(sync.Once)}
				cache[context] = miruken.LifestyleCache{key: entry}
			}
		} else {
			entry   = &miruken.LifestyleEntry{Once: new(sync.Once)}
			s.cache = map[*Context]miruken.LifestyleCache{context: {key: entry}}
		}
		s.lock.Unlock()
	}

	entry.Once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("scoped: panic: %v", r)
				}
				entry.Once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			entry.Once = new(sync.Once)
		} else {
			entry.Instance = out
			if contextual, ok := out[0].(Contextual); ok {
				contextual.SetContext(context)
				unsubscribe := contextual.Observe(s)
				context.Observe(EndedObserverFunc(func(*Context, any) {
					s.lock.Lock()
					delete(s.cache, context)
					s.lock.Unlock()
					unsubscribe.Dispose()
					s.tryDispose(out[0])
					contextual.SetContext(nil)
				}))
			} else {
				context.Observe(EndedObserverFunc(func(*Context, any) {
					s.lock.Lock()
					delete(s.cache, context)
					s.lock.Unlock()
					s.tryDispose(out[0])
				}))
			}
		}
	})
	return entry.Instance, nil, nil
}

func (s *scopedFilter) ContextChanging(
	contextual Contextual,
	oldCtx      *Context,
	newCtx     **Context,
) {
	if oldCtx == *newCtx {
		return
	}
	if *newCtx != nil {
		panic("managed instances cannot change context")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if cache := s.cache; cache == nil {
		return
	} else if keys := cache[oldCtx]; keys == nil {
		return
	} else {
		for key, entry := range keys {
			if entry.Instance[0] == contextual {
				delete(keys, key)
				s.tryDispose(contextual)
				break
			}
		}
	}
}

func (s *scopedFilter) isCompatibleWithParent(
	ctx miruken.HandleContext,
	rooted  bool,
) bool {
	if parent := ctx.Callback().(*provides.It).Parent(); parent != nil {
		if pb := parent.Binding(); pb != nil {
			for _, filter := range pb.Filters() {
				if scoped, ok := filter.(*scoped); !ok || (!rooted && scoped.rooted) {
					return false
				}
			}
		}
	}
	return true
}

func (s *scopedFilter) tryDispose(instance any) {
	if disposable, ok := instance.(miruken.Disposable); ok {
		disposable.Dispose()
	}
}

var (
	FromScope miruken.Qualifier[fromScope]
	contextType = miruken.TypeOf[*Context]()
)