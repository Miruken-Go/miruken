package context

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/provides"
	"sync"
	"sync/atomic"
)

type (
	// State represents the state of a Context.
	State uint

	// Reason identifies the cause for the notification.
	Reason uint

	// Contextual represents anything with a Context.
	Contextual interface {
		Context() *Context
		SetContext(*Context)
		Observe(Observer) miruken.Disposable
	}

	// A Context represents the scope at a give point in time.
	// Context has a beginning and an end and can handle callbacks As well As
	// notify observers of lifecycle changes.  In addition, it maintains
	// parent-child relationships and thus can form a graph.
	Context struct {
		miruken.MutableHandlers
		parent    *Context
		state     State
		children  slices.Safe[miruken.Traversing]
		observers atomic.Pointer[map[contextObserverType][]Observer]
		lock      sync.Mutex
	}

	// ContextualBase is a base implementation for Contextual.
	ContextualBase struct {
		ctx        *Context
		observers  map[contextualObserverType][]Observer
		lock       sync.RWMutex
	}
)


const (
	StateActive State = iota
	StateEnding
	StateEnded
)

const (
	ReasonAlreadyEnded Reason = iota
	ReasonUnwinded
	ReasonDisposed
)


// Context

func (c *Context) Parent() miruken.Traversing {
	return c.parent
}

func (c *Context) State() State {
	return c.state
}

func (c *Context) Children() []miruken.Traversing {
	return c.children.Items()
}

func (c *Context) Root() *Context {
	root := c
	for parent, _ := c.Parent().(*Context); parent != nil;
		parent, _ = parent.Parent().(*Context) {
		root = parent
	}
	return root
}

func (c *Context) HasChildren() bool {
	return len(c.children.Items()) > 0
}

func (c *Context) NewChild() *Context {
	c.ensureActive()
	child := &Context{
		parent: c,
		state:  StateActive,
	}
	child.observers.Store(&map[contextObserverType][]Observer{
		contextObserverEnding: {
			EndingObserverFunc(func(ctx *Context, reason any) {
				c.notify(contextObserverChildEnding, ctx, reason)
			})},
		contextObserverEnded: {
			EndedObserverFunc(func(ctx *Context, reason any) {
				c.removeChild(ctx)
				c.notify(contextObserverChildEnded, ctx, reason)
			})},
	})
	child.ResetHandlers(miruken.NewProvider(child))
	c.children.Append(child)
	return child
}

func (c *Context) Store(values ...any) *Context {
	providers := slices.Map[any, any](values, func (v any) any {
		return miruken.NewProvider(v)
	})
	c.AppendHandlers(providers...)
	return c
}

func (c *Context) Handle(
	callback any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if composer == nil {
		composer = &miruken.CompositionScope{Handler: c}
	}
	return c.MutableHandlers.Handle(callback, greedy, composer).
		OtherwiseIf(greedy, func () miruken.HandleResult {
			if parent := c.parent; parent != nil {
				return parent.Handle(callback, greedy, composer)
			}
			return miruken.NotHandled
		})
}

func (c *Context) HandleAxis(
	axis miruken.TraversingAxis,
	callback any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if composer == nil {
		composer = &miruken.CompositionScope{Handler: c}
	}
	if axis == miruken.TraverseSelf {
		return c.MutableHandlers.Handle(callback, greedy, composer)
	}
	result := miruken.NotHandled
	if err := miruken.TraverseAxis(c, axis, miruken.TraversalVisitorFunc(
		func(child miruken.Traversing) (bool, error) {
			if child == c {
				result = result.Or(c.MutableHandlers.Handle(callback, greedy, composer))
			} else if ctx, ok := child.(*Context); ok {
				result = result.Or(ctx.HandleAxis(miruken.TraverseSelf, callback, greedy, composer))
			}
			return result.Stop() || (result.Handled() && !greedy), nil
		})); err != nil {
		result = result.WithError(err)
	}
	return result
}

func (c *Context) Observe(observer Observer) miruken.Disposable {
	c.ensureActive()
	if observer == nil {
		return miruken.DisposableFunc(func() {})
	}
	var obsType contextObserverType
	if obs, ok := observer.(EndingObserver); ok {
		if c.state == StateEnding {
			obs.ContextEnding(c, ReasonAlreadyEnded)
		} else if c.state == StateActive {
			obsType |= contextObserverEnding
		}
	}
	if obs, ok := observer.(EndedObserver); ok {
		if c.state == StateEnded {
			obs.ContextEnded(c, ReasonAlreadyEnded)
		} else if c.state == StateActive {
			obsType |= contextObserverEnded
		}
	}
	if _, ok := observer.(ChildEndingObserver); ok {
		obsType |= contextObserverChildEnding
	}
	if _, ok := observer.(ChildEndedObserver); ok {
		obsType |= contextObserverChildEnded
	}
	c.addObserver(obsType, observer)
	return miruken.DisposableFunc(func() {
		c.removeObserver(obsType, observer)
	})
}

func (c *Context) Traverse(
	axis miruken.TraversingAxis,
	visitor miruken.TraversalVisitor,
) error {
	return miruken.TraverseAxis(c, axis, visitor)
}

func (c *Context) UnwindToRoot(reason any) *Context {
	return c.Root().Unwind(reason)
}

func (c *Context) Unwind(reason any) *Context {
	if internal.IsNil(reason) {
		reason = ReasonUnwinded
	}
	children := c.children.Items()
	for i := len(children)-1; i >= 0; i-- {
		children[i].(*Context).End(reason)
	}
	return c
}

func (c *Context) End(reason any) {
	if c.state != StateActive {
		return
	}
	c.state = StateEnding
	c.notify(contextObserverEnding, c, reason)
	defer func() {
		c.state = StateEnded
		c.notify(contextObserverEnded, c, reason)
	}()
	c.Unwind(nil)
}

func (c *Context) Dispose() {
	c.End(ReasonDisposed)
}

func (c *Context) removeChild(childCtx *Context) {
	c.children.Delete(func(t miruken.Traversing) (bool, bool) {
		match := t == childCtx
		return match, match
	})
}

func (c *Context) addObserver(
	obsType  contextObserverType,
	observer Observer,
) {
	if obsType == contextObserverNone {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	observers := c.observers.Load()
	var obs map[contextObserverType][]Observer
	if observers != nil {
		obs = make(map[contextObserverType][]Observer, len(*observers))
		for k, v := range *observers {
			os := make([]Observer, len(v))
			copy(os, v)
			obs[k] = os
		}
	} else {
		obs = map[contextObserverType][]Observer{}
	}
	for ot := contextObserverEnding; ot < contextObserverAll; ot <<= 1 {
		if obsType & ot == ot {
			obs[ot] = append(obs[ot], observer)
		}
	}
	c.observers.Store(&obs)
}

func (c *Context) removeObserver(
	obsType  contextObserverType,
	observer Observer,
) {
	if obsType == contextObserverNone {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	observers := c.observers.Load()
	if observers != nil {
		obs := make(map[contextObserverType][]Observer, len(*observers))
		for k, v := range *observers {
			var os []Observer
			if obsType & k == k {
				for i, o := range v {
					if o == observer {
						os = make([]Observer, len(v)-1)
						copy(os, v[:i])
						copy(os[i:], v[i+1:])
						obs[k] = os
						break
					}
				}
			}
			os = make([]Observer, len(v))
			copy(os, v)
			obs[k] = os
		}
		c.observers.Store(&obs)
	}
}

func (c *Context) notify(
	obsType  contextObserverType,
	ctx      *Context,
	reason   any,
) {
	if observers := c.observers.Load(); observers != nil {
		if obs, ok := (*observers)[obsType]; ok && len(obs) > 0 {
			switch obsType {
			case contextObserverEnding:
				for _, obs := range obs {
					obs.(EndingObserver).ContextEnding(ctx, reason)
				}
			case contextObserverEnded:
				for _, obs := range obs {
					obs.(EndedObserver).ContextEnded(ctx, reason)
				}
			case contextObserverChildEnding:
				for _, obs := range obs {
					obs.(ChildEndingObserver).ChildContextEnding(ctx, reason)
				}
			case contextObserverChildEnded:
				for _, obs := range obs {
					obs.(ChildEndedObserver).ChildContextEnded(ctx, reason)
				}
			}
		}
	}
}

func (c *Context) ensureActive() {
	if c.state != StateActive {
		panic("the context has already ended")
	}
}

func New(handlers ...any) *Context {
	context := &Context{
		state: StateActive,
	}
	context.ResetHandlers(append(handlers, miruken.NewProvider(context))...)
	return context
}


// ContextualBase

func (c *ContextualBase) Context() *Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ctx
}

func (c *ContextualBase) ChangeContext(
	contextual Contextual,
	ctx        *Context,
) {
	if contextual == nil {
		panic("contextual cannot be nil")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	oldCtx := c.ctx
	if ctx == oldCtx {
		return
	}
	newCtx := ctx
	c.notify(contextual, contextualObserverChanging, oldCtx, &newCtx)
	if oldCtx != nil {
		oldCtx.RemoveHandlers(contextual)
	}
	c.ctx = newCtx
	if newCtx != nil {
		newCtx.InsertHandlers(0, contextual)
	}
	c.notify(contextual, contextualObserverChanged, oldCtx, &newCtx)
}

func (c *ContextualBase) EndContext() {
	if ctx := c.Context(); ctx != nil {
		ctx.End(nil)
	}
}

func (c *ContextualBase) Observe(
	observer Observer,
) miruken.Disposable {
	if observer == nil {
		return miruken.DisposableFunc(func() {})
	}
	var obsType contextualObserverType
	if _, ok := observer.(ChangingObserver); ok {
		obsType |= contextualObserverChanging
	}
	if _, ok := observer.(ChangedObserver); ok {
		obsType |= contextualObserverChanged
	}
	c.addObserver(obsType, observer)
	return miruken.DisposableFunc(func() {
		c.removeObserver(obsType, observer)
	})
}

func (c *ContextualBase) addObserver(
	obsType contextualObserverType,
	observer Observer,
) {
	if obsType == contextualObserverNone {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.observers == nil {
		c.observers = make(map[contextualObserverType][]Observer)
	}
	for typ := contextualObserverChanging; typ < contextualObserverAll; typ <<= 1 {
		if obsType & typ == typ {
			c.observers[typ] = append(c.observers[typ], observer)
		}
	}
}

func (c *ContextualBase) removeObserver(
	obsType contextualObserverType,
	observer Observer,
) {
	if obsType == contextualObserverNone {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for typ := contextualObserverChanging; typ < contextualObserverAll; typ <<= 1 {
		if obsType & typ != typ {
			continue
		}
		if observers, ok := c.observers[typ]; ok && len(observers) > 0 {
			for i, obs := range observers {
				if obs == observer {
					copy(observers[i:], observers[i+1:])
					observers[len(observers)-1] = nil
					c.observers[typ] = observers[:len(observers)-1]
					break
				}
			}
		}
	}
}

func (c *ContextualBase) notify(
	contextual Contextual,
	obsType contextualObserverType,
	oldCtx    *Context,
	newCtx   **Context,
) {
	if observers, ok := c.observers[obsType]; ok && len(observers) > 0 {
		switch obsType {
		case contextualObserverChanging:
			for _, obs := range observers {
				obs.(ChangingObserver).ContextChanging(contextual, oldCtx, newCtx)
			}
		case contextualObserverChanged:
			for _, obs := range observers {
				obs.(ChangedObserver).ContextChanged(contextual, oldCtx, *newCtx)
			}
		}
	}
}


// Context observers

type (
	// Observer is a generic Context observer.
	Observer = any

	// EndingObserver reports Context is ending.
	EndingObserver interface {
		ContextEnding(ctx *Context, reason any)
	}
	EndingObserverFunc func(ctx *Context, reason any)

	// EndedObserver reports Context ended.
	EndedObserver interface {
		ContextEnded(ctx *Context, reason any)
	}
	EndedObserverFunc func(ctx *Context, reason any)

	// ChildEndingObserver reports child Context is ending.
	ChildEndingObserver interface {
		ChildContextEnding(childCtx *Context, reason any)
	}
	ChildEndingObserverFunc func(ctx *Context, reason any)

	// ChildEndedObserver reports child Context ended.
	ChildEndedObserver interface {
		ChildContextEnded(childCtx *Context, reason any)
	}
	ChildEndedObserverFunc func(ctx *Context, reason any)

	contextObserverType uint
)


const (
	contextObserverNone contextObserverType = 0
	contextObserverEnding contextObserverType = 1 << iota
	contextObserverEnded
	contextObserverChildEnding
	contextObserverChildEnded
	contextObserverAll = 1 << iota - 1
)


func (f EndingObserverFunc) ContextEnding(
	ctx    *Context,
	reason  any,
) {
	f(ctx, reason)
}

func (f EndedObserverFunc) ContextEnded(
	ctx    *Context,
	reason  any,
) {
	f(ctx, reason)
}

func (f ChildEndingObserverFunc) ChildContextEnding(
	ctx    *Context,
	reason  any,
) {
	f(ctx, reason)
}

func (f ChildEndedObserverFunc) ChildContextEnded(
	ctx    *Context,
	reason  any,
) {
	f(ctx, reason)
}


// Contextual observers

type (
	// ChangingObserver reports a Context is contextualObserverChanging.
	ChangingObserver interface {
		ContextChanging(
			contextual Contextual,
			oldCtx     *Context,
			newCtx     **Context)
	}
	ChangingObserverFunc func(
		contextual Contextual,
		oldCtx     *Context,
		newCtx     **Context)

	// ChangedObserver reports a Context contextualObserverChanged.
	ChangedObserver interface {
		ContextChanged(
			contextual Contextual,
			oldCtx     *Context,
			newCtx     *Context)
	}
	ChangedObserverFunc func(
		contextual Contextual,
		oldCtx     *Context,
		newCtx     *Context)

	contextualObserverType uint
)


const (
	contextualObserverNone contextualObserverType = 0
	contextualObserverChanging contextualObserverType = 1 << iota
	contextualObserverChanged
	contextualObserverAll = 1 << iota - 1
)


func (f ChangingObserverFunc) ContextChanging(
	contextual Contextual,
	oldCtx     *Context,
	newCtx     **Context,
) {
	f(contextual, oldCtx, newCtx)
}

func (f ChangedObserverFunc) ContextChanged(
	contextual Contextual,
	oldCtx     *Context,
	newCtx     *Context,
) {
	f(contextual, oldCtx, newCtx)
}


var PublishFromRoot miruken.BuilderFunc =
	 func (handler miruken.Handler) miruken.Handler {
		 if context, _, ok, err := provides.Type[*Context](handler); !(ok && err == nil) {
			 panic("root context could not be found")
		 } else {
			 return miruken.Publish.BuildUp(context.Root())
		 }
}
