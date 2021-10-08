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
}

// A Context represents the scope at a give point in time.
// It has a beginning and an end and can handle callbacks as well as
// notify observers of lifecycle changes.  In addition, it maintains
// parent-child relationships and thus can form a hierarchy.
type Context struct {
	handler    Handler
	parent    *Context
	state      ContextState
	children   []Traversing
	observers  map[observerType][]ContextObserver
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

func (c *Context) CreateChild() *Context {
	c.ensureActive()
	child := &Context{
		parent: c,
		state: ContextActive,
	}
	child.handler = NewProvider(child)
	child.Observe(ContextEndingObserverFunc(func (ctx *Context, reason interface{}) {
		c.notify(childContextEnding, ctx, reason)
	}))
	child.Observe(ContextEndedObserverFunc(func (ctx *Context, reason interface{}) {
		c.removeChild(ctx)
		c.notify(childContextEnded, ctx, reason)
	}))
	c.lock.Lock()
	defer c.lock.Unlock()
	c.children = append(c.children, child)
	return child
}

func (c *Context) Store(values ... interface{}) *Context {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.handler = Build(c.handler, With(values...))
	return c
}

func (c *Context) AddHandlers(
	handlers ... interface{},
) *Context {
	if len(handlers) > 0 {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.handler = AddHandlers(c.handler, handlers...)
	}
	return c
}

func (c *Context) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	if handler := c.handler; handler != nil {
		return handler.Handle(callback, greedy, composer).
			OtherwiseIf(greedy, func (HandleResult) HandleResult {
				if parent := c.parent; parent != nil {
					return parent.Handle(callback, greedy, composer)
				}
				return NotHandled
			})
	}
	return NotHandled
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
		if handler := c.handler; handler != nil {
			return handler.Handle(callback, greedy, composer)
		}
	}
	result := NotHandled
	if err := TraverseAxis(c, axis, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			if child == c {
				if handler := c.handler; handler != nil {
					result = result.Or(handler.Handle(callback, greedy, composer))
				}
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
	var obsType observerType
	if obs, ok := observer.(ContextEndingObserver); ok {
		if c.state == ContextEnding {
			obs.ContextEnding(c, ContextAlreadyEnded)
		} else if c.state == ContextActive {
			obsType |= contextEnding
		}
	}
	if obs, ok := observer.(ContextEndedObserver); ok {
		if c.state == ContextEnded {
			obs.ContextEnded(c, ContextAlreadyEnded)
		} else if c.state == ContextActive {
			obsType |= contextEnded
		}
	}
	if _, ok := observer.(ChildContextEndingObserver); ok {
		obsType |= childContextEnding
	}
	if _, ok := observer.(ChildContextEndedObserver); ok {
		obsType |= childContextEnded
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
	c.notify(contextEnding, c, reason)
	defer func() {
		c.state = ContextEnded
		c.notify(contextEnded, c, reason)
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
	obsType  observerType,
	observer interface{},
) {
	if obsType == noObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.observers == nil {
		c.observers = make(map[observerType][]ContextObserver)
	}
	for typ := contextEnding; typ < allObservers; typ <<= 1 {
		if obsType & typ == typ {
			c.observers[typ] = append(c.observers[typ], observer)
		}
	}
}

func (c *Context) removeObserver(
	obsType  observerType,
	observer interface{},
) {
	if obsType == noObservers {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for typ := contextEnding; typ < allObservers; typ <<= 1 {
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
	obsType  observerType,
	ctx     *Context,
	reason   interface{},
) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if observers, ok := c.observers[obsType]; ok && len(observers) > 0 {
		switch obsType {
		case contextEnding:
			for _, obs := range observers {
				obs.(ContextEndingObserver).ContextEnding(ctx, reason)
			}
		case contextEnded:
			for _, obs := range observers {
				obs.(ContextEndedObserver).ContextEnded(ctx, reason)
			}
		case childContextEnding:
			for _, obs := range observers {
				obs.(ChildContextEndingObserver).ChildContextEnding(ctx, reason)
			}
		case childContextEnded:
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
		state:   ContextActive,
		handler: NewRootHandler(builders...),
	}
	return context.AddHandlers(NewProvider(context))
}

type observerType uint

const (
	noObservers = 0
	contextEnding observerType = 1 << iota
	contextEnded
	childContextEnding
	childContextEnded
	allObservers = 1 << iota - 1
)

type (
	// ContextObserver is a generic observer.
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

var WithPublishFromRoot BuilderFunc = func (handler Handler) Handler {
	var context *Context
	if err := Resolve(handler, &context); err != nil {
		panic("the root context could not be found")
	}
	return WithPublish.Build(context.Root())
}
