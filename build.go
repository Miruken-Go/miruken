package miruken

import (
	"github.com/miruken-go/miruken/slices"
)

type (
	// Builder augments a Handler.
	Builder interface {
		BuildUp(handler Handler) Handler
	}

	// BuilderFunc promotes a function to Builder.
	BuilderFunc func(Handler) Handler
)


func (f BuilderFunc) BuildUp(
	handler Handler,
) Handler { return f(handler) }


func BuildUp(handler Handler, builders ...Builder) Handler {
	for _, b := range builders {
		if b != nil {
			handler = b.BuildUp(handler)
		}
	}
	return handler
}

func AddHandlers(
	parent   Handler,
	handlers ...any,
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

func ComposeBuilders(builder Builder, builders ...Builder) Builder {
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

func PipeBuilders(builder Builder, builders ...Builder) Builder {
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


// MutableHandlers manages a mutable list of Handlers.
type MutableHandlers struct {
	handlers slices.Safe[Handler]
}

func (m *MutableHandlers) Handlers() []any {
	return slices.Map[Handler, any](m.handlers.Items(), func(h Handler) any {
		if a, ok := h.(handlerAdapter); ok {
			return a.handler
		}
		return h
	})
}

func (m *MutableHandlers) ResetHandlers(
	handlers ...any,
) *MutableHandlers {
	m.handlers.Reset(normalizeHandlers(handlers)...)
	return m
}

func (m *MutableHandlers) AppendHandlers(
	handlers ...any,
) *MutableHandlers {
	if len(handlers) > 0 {
		m.handlers.Append(normalizeHandlers(handlers)...)
	}
	return m
}

func (m *MutableHandlers) InsertHandlers(
	index    int,
	handlers ...any,
) *MutableHandlers {
	if index < 0 {
		panic("index must be >= 0")
	}
	if len(handlers) > 0 {
		m.handlers.Insert(index, normalizeHandlers(handlers)...)
	}
	return m
}

func (m *MutableHandlers) RemoveHandlers(
	handlers ...any,
) *MutableHandlers {
	if len(handlers) > 0 {
		m.handlers.Delete(func(h Handler) (bool, bool) {
			for _, ht := range handlers {
				if h == ht {
					return true, false
				}
				if a, ok := h.(handlerAdapter); ok {
					return a.handler == ht, false
				}
			}
			return false, false
		})
	}
	return m
}

func (m *MutableHandlers) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, m)

	result := NotHandled
	for _, h := range m.handlers.Items() {
		if result.stop || (result.handled && !greedy) {
			return result
		}
		result = result.Or(h.Handle(callback, greedy, composer))
	}
	return result
}

func (m *MutableHandlers) SuppressDispatch() {}


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
		*incoming = &CompositionScope{receiver}
	}
}

func normalizeHandlers(handlers []any) []Handler {
	hs := make([]Handler, len(handlers))
	for i, v := range handlers {
		hs[i] = ToHandler(v)
	}
	return hs
}
