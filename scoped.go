package miruken

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sync"
)

// Scoped LifestyleProvider provides instances per Context.
type Scoped struct {
	LifestyleProvider
	rooted bool
}

func (s *Scoped) InitWithTag(tag reflect.StructTag) error {
	if scoped, ok := tag.Lookup("scoped"); ok {
		s.rooted = scoped == "rooted"
	}
	return s.Init()
}

func (s *Scoped) Init() error {
	s.SetFilters(&scoped{}, _constraintFilter[0])
	return nil
}

func (s *Scoped) Constraint() BindingConstraint {
	return Qualifier[Scoped]{}
}

// scoped is a Filter that caches an instance per Context.
type scoped struct {
	Lifestyle
	cache  map[*Context]lifestyleCache
	lock   sync.RWMutex
}

func (s *scoped) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, po *promise.Promise[[]any], err error) {
	key := ctx.Callback().(*Provides).Key()
	if key == _contextType {
		// can't resolve a context contextually
		return nil, nil,nil
	}

	rooted := false
	if scp, ok := provider.(*Scoped); ok {
		rooted = scp.rooted
	}

	if !s.isCompatibleWithParent(ctx, rooted) {
		return nil, nil,nil
	}
	context, _, err := Resolve[*Context](ctx.Composer())
	if err != nil {
		return nil, nil, err
	} else if context == nil {
		return next.Abort()
	} else if context.State() != ContextActive {
		return nil, nil, errors.New("scoped: cannot scope instances to an inactive context")
	} else if rooted {
		context = context.Root()
	}

	var entry *lifestyleEntry
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
					entry     = &lifestyleEntry{once: new(sync.Once)}
					keys[key] = entry
				}
			} else {
				entry = &lifestyleEntry{once: new(sync.Once)}
				cache[context] = lifestyleCache{key: entry}
			}
		} else {
			entry   = &lifestyleEntry{once: new(sync.Once)}
			s.cache = map[*Context]lifestyleCache{context: {key: entry}}
		}
		s.lock.Unlock()
	}

	entry.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("singleton: panic: %v", r)
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
				context.Observe(ContextEndedObserverFunc(func(*Context, any) {
					s.lock.Lock()
					delete(s.cache, context)
					s.lock.Unlock()
					unsubscribe.Dispose()
					s.tryDispose(out[0])
					contextual.SetContext(nil)
				}))
			} else {
				context.Observe(ContextEndedObserverFunc(func(*Context, any) {
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

func (s *scoped) ContextChanging(
	contextual   Contextual,
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

func (s *scoped) isCompatibleWithParent(
	ctx    HandleContext,
	rooted  bool,
) bool {
	if parent := ctx.Callback().(*Provides).Parent(); parent != nil {
		if pb := parent.Binding(); pb != nil {
			for _, filter := range pb.Filters() {
				if scoped, ok := filter.(*Scoped); !ok || (!rooted && scoped.rooted) {
					return false
				}
			}
		}
	}
	return true
}

func (s *scoped) tryDispose(instance any) {
	if disposable, ok := instance.(Disposable); ok {
		disposable.Dispose()
	}
}
