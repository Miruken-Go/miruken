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
	// Lifestyle associates both the scoped lifestyle and fromScope constraint.
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

	// scopedEntry stores a lazy instance.
	scopedEntry struct {
		instance []any
		once     *sync.Once
	}

	// scopedCache maintains a cache of scopedEntry's.
	scopedCache map[any]*scopedEntry

	// fromScope is a constraint for the context Lifestyle.
	fromScope struct {}
)


// From constrains resolution to a handler with scoped lifestyle.
// This is used to suppress resolving implied values available through a Context.
var From miruken.Qualifier[fromScope]


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
	cache  map[*Context]scopedCache
	lock   sync.RWMutex
}

func (s *scopedFilter) Next(
	_        miruken.Filter,
	next     miruken.Next,
	ctx      miruken.HandleContext,
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
	context, _, err := provides.Type[*Context](ctx.Composer())
	if err != nil {
		return nil, nil, err
	} else if context == nil {
		return next.Abort()
	} else if context.State() != StateActive {
		return nil, nil, errors.New("scoped: cannot scope instances to an inactive context")
	} else if rooted {
		context = context.Root()
	}

	var entry *scopedEntry
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
					if typ, ok := key.(reflect.Type); ok {
						for _,v := range keys {
							if instance := v.instance; len(instance) > 0 {
								if o := instance[0]; o != nil && reflect.TypeOf(o).AssignableTo(typ) {
									entry = v
									keys[key] = v
									break
								}
							}
						}
					}
					if entry == nil {
						entry = &scopedEntry{once: new(sync.Once)}
						keys[key] = entry
					}
				}
			} else {
				entry = &scopedEntry{once: new(sync.Once)}
				cache[context] = scopedCache{key: entry}
			}
		} else {
			entry   = &scopedEntry{once: new(sync.Once)}
			s.cache = map[*Context]scopedCache{context: {key: entry}}
		}
		s.lock.Unlock()
	}

	entry.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("scoped: panic: %v", r)
				}
				entry.once = new(sync.Once)
			}
		}()
		if out, po, err = next.Pipe(); err == nil && po != nil {
			out, err = po.Await()
		}
		if err != nil || len(out) == 0 {
			entry.once = new(sync.Once)
		} else {
			entry.instance = out
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
	return entry.instance, nil, nil
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
			if entry.instance[0] == contextual {
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


var contextType = miruken.TypeOf[*Context]()
