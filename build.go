package miruken

import (
	"github.com/miruken-go/miruken/slices"
	"sync"
)

type (
	// Builder augments a Handler.
	Builder interface {
		BuildUp(handler Handler) Handler
	}
	BuilderFunc func(Handler) Handler
)

func (f BuilderFunc) BuildUp(
	handler Handler,
) Handler { return f(handler) }

func composeBuilder2(builder1, builder2 Builder) Builder {
	if builder1 == nil {
		return builder2
	} else if builder2 == nil {
		return builder1
	}
	return BuilderFunc(func(handler Handler) Handler {
		return builder1.BuildUp(builder2.BuildUp(handler))
	})
}

func ComposeBuilders(builder Builder, builders ... Builder) Builder {
	switch len(builders) {
	case 0: return builder
	case 1: return composeBuilder2(builder, builders[0])
	default:
		for _, b := range builders {
			builder = composeBuilder2(builder, b)
		}
		return builder
	}
}

func PipeBuilders(builder Builder, builders ... Builder) Builder {
	switch len(builders) {
	case 0: return builder
	case 1: return composeBuilder2(builders[0], builder)
	default:
		b := builders[len(builders)-1]
		for i := len(builders)-2; i >= 0; i-- {
			b = composeBuilder2(b, builders[i])
		}
		return composeBuilder2(b, builder)
	}
}

func BuildUp(handler Handler, builders ... Builder) Handler {
	for _, b := range builders {
		if b != nil {
			handler = b.BuildUp(handler)
		}
	}
	return handler
}

func AddHandlers(
	parent Handler,
	handlers ... any,
) Handler {
	if parent == nil {
		panic("cannot add handlers to a nil parent")
	}

	hs := normalizeHandlers(handlers)

	switch c := len(hs); {
	case c == 1:
		return &withHandler{parent, hs[0]}
	case c > 1:
		return &withHandlers{parent, hs}
	default:
		return parent
	}
}

func With(values ... any) BuilderFunc {
	return func (handler Handler) Handler {
		var valueHandlers []any
		for _, val := range values {
			if val != nil {
				valueHandlers = append(valueHandlers, NewProvider(val))
			}
		}
		if len(valueHandlers) > 0 {
			return AddHandlers(handler, valueHandlers...)
		}
		return handler
	}
}

// withHandler composes two Handlers.
type withHandler struct {
	Handler
	handler Handler
}

func (w *withHandler) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, w)
	return w.handler.Handle(callback, greedy, composer).
		OtherwiseIf(greedy, func () HandleResult {
			return w.Handler.Handle(callback, greedy, composer)
		})
}

func (w *withHandler) SuppressDispatch() {}

// withHandlers composes any number of Handlers.
type withHandlers struct {
	Handler
	handlers []Handler
}

func (w *withHandlers) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, w)

	result := NotHandled

	for _, h := range w.handlers {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, composer))
	}
	return result.OtherwiseIf(greedy, func () HandleResult {
		return w.Handler.Handle(callback, greedy, composer)
	})
}

func (w *withHandlers) SuppressDispatch() {}

// mutableHandlers manages any number of Handlers.
type mutableHandlers struct {
	handlers []Handler
	lock     sync.RWMutex
}

func (m *mutableHandlers) Handlers() []any {
	return slices.Map[Handler, any](m.handlers, func(handler Handler) any {
		if a, ok := handler.(handlerAdapter); ok {
			return a.handler
		}
		return handler
	})
}

func (m *mutableHandlers) AddHandlers(
	handlers ... any,
) *mutableHandlers {
	if len(handlers) > 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		hs := normalizeHandlers(handlers)
		m.handlers = append(m.handlers, hs...)
	}
	return m
}

func (m *mutableHandlers) InsertHandlers(
	index        int,
	handlers ... any,
) *mutableHandlers {
	if index < 0 {
		panic("index must be >= 0")
	}
	if len(handlers) > 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		hs := normalizeHandlers(handlers)
		m.handlers = append(hs, m.handlers...)
	}
	return m
}

func (m *mutableHandlers) RemoveHandlers(
	handlers ... any,
) *mutableHandlers {
	if len(handlers) > 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		if len(m.handlers) > 0 {
			for i := len(m.handlers)-1; i >= 0; i-- {
				for _, h := range handlers {
					handler := m.handlers[i]
					if handler != h {
						if a, ok := handler.(handlerAdapter); !ok || a.handler != h {
							continue
						}
					}
					m.handlers = append(m.handlers[:i], m.handlers[i+1:]...)
					break
				}
			}
		}
	}
	return m
}

func (m *mutableHandlers) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, m)

	result := NotHandled

	if handlers := m.handlers; len(handlers) > 0 {
		for _, h := range m.handlers {
			if result.stop || (result.handled && !greedy) {
				return result
			}
			result = result.Or(h.Handle(callback, greedy, composer))
		}
	}

	return result
}

func (m *mutableHandlers) SuppressDispatch() {}

type (
	// ProceedFunc calls the next filter in the pipeline.
	ProceedFunc func() HandleResult

	// FilterFunc defines a function that can intercept a callback.
	FilterFunc func(
		callback any,
		greedy   bool,
		composer Handler,
		proceed  ProceedFunc,
	) HandleResult

	// filterHandler applies a filter to a Handler.
	filterHandler struct {
		Handler
		filter    FilterFunc
		reentrant bool
	}
)

func (f *filterHandler) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, f)

	if !f.reentrant {
		if _, ok := callback.(*Composition); ok {
			return f.Handler.Handle(callback, greedy, composer)
		}
	}
	return f.filter(callback, greedy, composer, func() HandleResult {
		return f.Handler.Handle(callback, greedy, composer)
	})
}

func (f FilterFunc) BuildUp(handler Handler) Handler {
	if f == nil { return handler }
	return &filterHandler{handler, f, false}
}

func Reentrant(filter FilterFunc) BuilderFunc {
	if filter == nil {
		panic("filter cannot be nil")
	}
	return func (handler Handler) Handler {
		return &filterHandler{handler, filter, true}
	}
}

func tryInitializeComposer(
	incoming *Handler,
	receiver  Handler,
) {
	if *incoming == nil {
		*incoming = &compositionScope{receiver}
	}
}

func normalizeHandlers(handlers []any) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}
