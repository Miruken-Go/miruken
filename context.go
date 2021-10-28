package miruken

import (
	"sync"
)

// ContextState represents the state of a Context.
type ContextState uint

const (
	ContextActive ContextState = iota
	ContextEnding
	ContextEnded
)

// ContextReason identifies the cause for the notification.
type ContextReason uint

const (
	ContextAlreadyEnded ContextReason = iota
	ContextUnwinded
	ContextDisposed
)

// Contextual represents anything with a Context.
type Contextual interface {
	Context() *Context
	SetContext(ctx *Context)
    Observe(observer ContextObserver) Disposable
}

// A Context represents the scope at a give point in time.
// It has a beginning and an end and can handle callbacks as well as
// notify observers of lifecycle changes.  In addition, it maintains
// parent-child relationships and thus can form a graph.
type Context struct {
	mutableHandlers
	parent    *Context
	state      ContextState
	children   []Traversing
	observers  map[ctxObserverType][]ContextObserver
	lock       sync.RWMutex
}

func (c *Context) Parent() Traversing {
	return c.parent
}

func (c *Context) State() ContextState {
	return c.state
}

func (c *Context) Children() []Traversing {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.children
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
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.children) > 0
}

func (c *Context) NewChild() *Context {
	c.ensureActive()
	child := &Context{
		parent: c,
		state: ContextActive,
	}
	child.AddHandlers(NewProvider(child))
	child.Observe(ContextEndingObserverFunc(func (ctx *Context, reason interface{}) {
		c.notify(childCtxEnding, ctx, reason)
	}))
	child.Observe(ContextEndedObserverFunc(func (ctx *Context, reason interface{}) {
		c.removeChild(ctx)
		c.notify(childCtxEnded, ctx, reason)
	}))
	c.lock.Lock()
	defer c.lock.Unlock()
	c.children = append(c.children, child)
	return child
}

func (c *Context) Store(values ... interface{}) *Context {
	for _, val := range values {
		c.AddHandlers(NewProvider(val))
	}
	return c
}

func (c *Context) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if composer == nil {
		composer = &compositionScope{c}
	}
	return c.mutableHandlers.Handle(callback, greedy, composer).
		OtherwiseIf(greedy, func (HandleResult) HandleResult {
			if parent := c.parent; parent != nil {
				return parent.Handle(callback, greedy, composer)
			}
			return NotHandled
		})
}

func (c *Context) HandleAxis(
	axis     TraversingAxis,
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if composer == nil {
		composer = &compositionScope{c}
	}
	if axis == TraverseSelf {
		return c.mutableHandlers.Handle(callback, greedy, composer)
	}
	result := NotHandled
	if err := TraverseAxis(c, axis, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			if child == c {
				result = result.Or(c.mutableHandlers.Handle(callback, greedy, composer))
			} else if ctx, ok := child.(*Context); ok {
				result = result.Or(ctx.HandleAxis(TraverseSelf, callback, greedy, composer))
			}
			return result.ShouldStop() || (result.IsHandled() && !greedy), nil
		})); err != nil {
		result = result.WithError(err)
	}
	return result
}

func (c *Context) Observe(observer ContextObserver) Disposable {
	c.ensureActive()
	if observer == nil {
		return DisposableFunc(func() {})
	}
	var obsType ctxObserverType
	if obs, ok := observer.(ContextEndingObserver); ok {
		if c.state == ContextEnding {
			obs.ContextEnding(c, ContextAlreadyEnded)
		} else if c.state == ContextActive {
			obsType |= ctxEnding
		}
	}
	if obs, ok := observer.(ContextEndedObserver); ok {
		if c.state == ContextEnded {
			obs.ContextEnded(c, ContextAlreadyEnded)
		} else if c.state == ContextActive {
			obsType |= ctxEnded
		}
	}
	if _, ok := observer.(ChildContextEndingObserver); ok {
		obsType |= childCtxEnding
	}
	if _, ok := observer.(ChildContextEndedObserver); ok {
		obsType |= childCtxEnded
	}
	c.addObserver(obsType, observer)
	return DisposableFunc(func() {
		c.removeObserver(obsType, observer)
	})
}

func (c *Context) Traverse(
	axis    TraversingAxis,
	visitor TraversalVisitor,
) error {
	return TraverseAxis(c, axis, visitor)
}

func (c *Context) UnwindToRoot(reason interface{}) *Context {
	return c.Root().Unwind(reason)
}

func (c *Context) Unwind(reason interface{}) *Context {
	if reason == nil {
		reason = ContextUnwinded
	}
	for i := len(c.children)-1; i >= 0; i-- {
		c.children[i].(*Context).End(nil)
	}
	return c
}

func (c *Context) End(reason interface{}) {
	if c.state != ContextActive {
		return
	}
	c.state = ContextEnding
	c.notify(ctxEnding, c, reason)
	defer func() {
		c.state = ContextEnded
		c.notify(ctxEnded, c, reason)
	}()
	c.Unwind(nil)
}

func (c *Context) Dispose() {
	c.End(ContextDisposed)
}

func (c *Context) removeChild(childCtx *Context) {
	c.lock.Lock()
	defer c.lock.Unlock()
	children := c.children
	for i, child := range children {
		if child == childCtx {
			copy(children[i:], children[i+1:])
			children[len(children)-1] = nil
			c.children = children[:len(children)-1]
			break
		}
	}
}

func (c *Context) addObserver(
	obsType  ctxObserverType,
	observer ContextObserver,
) {
	if obsType == noCtxObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.observers == nil {
		c.observers = make(map[ctxObserverType][]ContextObserver)
	}
	for typ := ctxEnding; typ < allCtxObservers; typ <<= 1 {
		if obsType & typ == typ {
			c.observers[typ] = append(c.observers[typ], observer)
		}
	}
}

func (c *Context) removeObserver(
	obsType  ctxObserverType,
	observer ContextObserver,
) {
	if obsType == noCtxObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for typ := ctxEnding; typ < allCtxObservers; typ <<= 1 {
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

func (c *Context) notify(
	obsType  ctxObserverType,
	ctx     *Context,
	reason   interface{},
) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if observers, ok := c.observers[obsType]; ok && len(observers) > 0 {
		switch obsType {
		case ctxEnding:
			for _, obs := range observers {
				obs.(ContextEndingObserver).ContextEnding(ctx, reason)
			}
		case ctxEnded:
			for _, obs := range observers {
				obs.(ContextEndedObserver).ContextEnded(ctx, reason)
			}
		case childCtxEnding:
			for _, obs := range observers {
				obs.(ChildContextEndingObserver).ChildContextEnding(ctx, reason)
			}
		case childCtxEnded:
			for _, obs := range observers {
				obs.(ChildContextEndedObserver).ChildContextEnded(ctx, reason)
			}
		}
	}
}

func (c *Context) ensureActive() {
	if c.state != ContextActive {
		panic("the context has already ended")
	}
}

func NewContext(builders ... Builder) *Context {
	context := &Context{
		state:           ContextActive,
		mutableHandlers: mutableHandlers{parent: NewRootHandler(builders...)},
	}
	context.AddHandlers(NewProvider(context))
	return context
}

type ContextualBase struct {
	ctx        *Context
	observers   map[contextualObserverType][]ContextObserver
	lock        sync.RWMutex
}

func (c *ContextualBase) Context() *Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ctx
}

func (c *ContextualBase) ChangeContext(
	contextual  Contextual,
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
	c.notify(contextual, ctxChanging, oldCtx, &newCtx)
	if oldCtx != nil {
		oldCtx.RemoveHandlers(contextual)
	}
	c.ctx = newCtx
	if newCtx != nil {
		newCtx.InsertHandlers(0, contextual)
	}
	c.notify(contextual, ctxChanged, oldCtx, &newCtx)
}

func (c *ContextualBase) EndContext() {
	if ctx := c.Context(); ctx != nil {
		ctx.End(nil)
	}
}

func (c *ContextualBase) Observe(
	observer ContextObserver,
) Disposable {
	if observer == nil {
		return DisposableFunc(func() {})
	}
	var obsType contextualObserverType
	if _, ok := observer.(ContextChangingObserver); ok {
		obsType |= ctxChanging
	}
	if _, ok := observer.(ContextChangedObserver); ok {
		obsType |= ctxChanged
	}
	c.addObserver(obsType, observer)
	return DisposableFunc(func() {
		c.removeObserver(obsType, observer)
	})
}

func (c *ContextualBase) addObserver(
	obsType  contextualObserverType,
	observer ContextObserver,
) {
	if obsType == noContextualObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.observers == nil {
		c.observers = make(map[contextualObserverType][]ContextObserver)
	}
	for typ := ctxChanging; typ < allContextualObservers; typ <<= 1 {
		if obsType & typ == typ {
			c.observers[typ] = append(c.observers[typ], observer)
		}
	}
}

func (c *ContextualBase) removeObserver(
	obsType  contextualObserverType,
	observer ContextObserver,
) {
	if obsType == noContextualObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for typ := ctxChanging; typ < allContextualObservers; typ <<= 1 {
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
	obsType    contextualObserverType,
	oldCtx    *Context,
	newCtx   **Context,
) {
	if observers, ok := c.observers[obsType]; ok && len(observers) > 0 {
		switch obsType {
		case ctxChanging:
			for _, obs := range observers {
				obs.(ContextChangingObserver).ContextChanging(contextual, oldCtx, newCtx)
			}
		case ctxChanged:
			for _, obs := range observers {
				obs.(ContextChangedObserver).ContextChanged(contextual, oldCtx, *newCtx)
			}
		}
	}
}

// Context observers

type ctxObserverType uint

const (
	noCtxObservers ctxObserverType = 0
	ctxEnding ctxObserverType = 1 << iota
	ctxEnded
	childCtxEnding
	childCtxEnded
	allCtxObservers = 1 << iota - 1
)

type (
	// ContextObserver is a generic Context observer.
	ContextObserver = interface {}

	// ContextEndingObserver reports Context is ending.
	ContextEndingObserver interface {
		ContextEnding(ctx *Context, reason interface{})
	}
	ContextEndingObserverFunc func(ctx *Context, reason interface{})

	// ContextEndedObserver reports Context ended.
	ContextEndedObserver interface {
		ContextEnded(ctx *Context, reason interface{})
	}
	ContextEndedObserverFunc func(ctx *Context, reason interface{})

	// ChildContextEndingObserver reports child Context is ending.
	ChildContextEndingObserver interface {
		ChildContextEnding(childCtx *Context, reason interface{})
	}
	ChildContextEndingObserverFunc func(ctx *Context, reason interface{})

	// ChildContextEndedObserver reports child Context ended.
	ChildContextEndedObserver interface {
		ChildContextEnded(childCtx *Context, reason interface{})
	}
	ChildContextEndedObserverFunc func(ctx *Context, reason interface{})
)

func (f ContextEndingObserverFunc) ContextEnding(
	ctx    *Context,
	reason  interface{},
) {
	f(ctx, reason)
}

func (f ContextEndedObserverFunc) ContextEnded(
	ctx    *Context,
	reason  interface{},
) {
	f(ctx, reason)
}

func (f ChildContextEndingObserverFunc) ChildContextEnding(
	ctx    *Context,
	reason  interface{},
) {
	f(ctx, reason)
}

func (f ChildContextEndedObserverFunc) ChildContextEnded(
	ctx    *Context,
	reason  interface{},
) {
	f(ctx, reason)
}

// Contextual observers

type contextualObserverType uint

const (
	noContextualObservers contextualObserverType = 0
	ctxChanging contextualObserverType = 1 << iota
	ctxChanged
	allContextualObservers = 1 << iota - 1
)

type (
	// ContextChangingObserver reports a Context is changing.
	ContextChangingObserver interface {
		ContextChanging(
			contextual   Contextual,
			oldCtx      *Context,
			newCtx     **Context)
	}
	ContextChangingObserverFunc func(
		contextual   Contextual,
		oldCtx      *Context,
		newCtx     **Context)

	// ContextChangedObserver reports a Context changed.
	ContextChangedObserver interface {
		ContextChanged(
			contextual  Contextual,
			oldCtx     *Context,
			newCtx     *Context)
	}
	ContextChangedObserverFunc func(
		contextual  Contextual,
		oldCtx     *Context,
		newCtx     *Context)
)

func (f ContextChangingObserverFunc) ContextChanging(
	contextual   Contextual,
	oldCtx      *Context,
	newCtx     **Context,
) {
	f(contextual, oldCtx, newCtx)
}

func (f ContextChangedObserverFunc) ContextChanged(
	contextual  Contextual,
	oldCtx     *Context,
	newCtx     *Context,
) {
	f(contextual, oldCtx, newCtx)
}

var WithPublishFromRoot BuilderFunc = func (handler Handler) Handler {
	var context *Context
	if err := Resolve(handler, &context); err != nil {
		panic("the root context could not be found")
	}
	return WithPublish.Build(context.Root())
}
