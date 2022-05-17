package miruken

import (
	"errors"
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
	return ScopedQualifier{}
}

// scoped is a Filter that caches an instance per Context.
type scoped struct {
	Lifestyle
	Qualifier
	cache  map[*Context][]any
	lock   sync.RWMutex
}

func (s *scoped) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  ([]any, error) {
	rooted := false
	if scp, ok := provider.(*Scoped); ok {
		rooted = scp.rooted
	}
	if !s.isCompatibleWithParent(context, rooted) {
		return nil, nil
	}
	ctx, err := Resolve[*Context](context.Composer())
	if err != nil {
		return nil, err
	} else if ctx == nil {
		return next.Abort()
	} else if ctx.State() != ContextActive {
		return nil, errors.New("cannot scope instances to an inactive context")
	} else if rooted {
		ctx = ctx.Root()
	}
	var instance []any
	s.lock.RLock()
	if s.cache != nil {
		instance = s.cache[ctx]
		if instance != nil {
			defer s.lock.RUnlock()
			return instance, nil
		}
	}
	s.lock.RUnlock()
	if res, err := next.Filter(); err != nil || len(res) == 0 {
		return res, err
	} else {
		s.lock.Lock()
		if s.cache != nil {
			if instance = s.cache[ctx]; instance != nil {
				defer s.lock.Unlock()
				return instance, nil
			}
		} else {
			s.cache = map[*Context][]any{}
		}
		instance     = res
		s.cache[ctx] = res
		s.lock.Unlock()
	}
	if contextual, ok := instance[0].(Contextual); ok {
		contextual.SetContext(ctx)
		unsubscribe := contextual.Observe(s)
		ctx.Observe(ContextEndedObserverFunc(func(*Context, any) {
			s.lock.Lock()
			delete(s.cache, ctx)
			s.lock.Unlock()
			unsubscribe.Dispose()
			s.tryDispose(instance[0])
			contextual.SetContext(nil)
		}))
	} else {
		ctx.Observe(ContextEndedObserverFunc(func(*Context, any) {
			s.lock.Lock()
			delete(s.cache, ctx)
			s.lock.Unlock()
			s.tryDispose(instance[0])
		}))
	}
	return instance, nil
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
	if instance, ok := s.cache[oldCtx]; !ok || contextual != instance[0] {
		return
	}
	delete(s.cache, oldCtx)
	s.tryDispose(contextual)
}

func (s *scoped) isCompatibleWithParent(
	context  HandleContext,
	rooted   bool,
) bool {
	if parent := context.Callback().(*Provides).Parent(); parent != nil {
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

// ScopedQualifier constrains a scoped lifestyle.
type ScopedQualifier struct {
	Qualifier
}

func (s ScopedQualifier) Require(metadata *BindingMetadata) {
	s.RequireQualifier(s, metadata)
}

func (s ScopedQualifier) Matches(metadata *BindingMetadata) bool {
	return s.MatchesQualifier(s, metadata)
}
