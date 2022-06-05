package miruken

import (
	"errors"
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
	cache  map[*Context][]any
	lock   sync.RWMutex
}

func (s *scoped) Next(
	next     Next,
	ctx      HandleContext,
	provider FilterProvider,
)  ([]any, *promise.Promise[[]any], error) {
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
	var instance []any
	s.lock.RLock()
	if s.cache != nil {
		instance = s.cache[context]
		if instance != nil {
			defer s.lock.RUnlock()
			return instance, nil, nil
		}
	}
	s.lock.RUnlock()
	res, pr, err := next.Pipe()
	if err == nil && pr != nil {
		res, err = pr.Await()
	}
	if err != nil || len(res) == 0 {
		return res, nil, err
	} else {
		s.lock.Lock()
		if s.cache != nil {
			if instance = s.cache[context]; instance != nil {
				defer s.lock.Unlock()
				return instance, nil, nil
			}
		} else {
			s.cache = map[*Context][]any{}
		}
		instance     = res
		s.cache[context] = res
		s.lock.Unlock()
	}
	if contextual, ok := instance[0].(Contextual); ok {
		contextual.SetContext(context)
		unsubscribe := contextual.Observe(s)
		context.Observe(ContextEndedObserverFunc(func(*Context, any) {
			s.lock.Lock()
			delete(s.cache, context)
			s.lock.Unlock()
			unsubscribe.Dispose()
			s.tryDispose(instance[0])
			contextual.SetContext(nil)
		}))
	} else {
		context.Observe(ContextEndedObserverFunc(func(*Context, any) {
			s.lock.Lock()
			delete(s.cache, context)
			s.lock.Unlock()
			s.tryDispose(instance[0])
		}))
	}
	return instance, nil, nil
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
