package miruken

import (
	"context"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sync/atomic"
)

type (
	batching interface {
		CompleteBatch(
			composer Handler,
		) (any, *promise.Promise[any], error)
	}

	batch struct {
		MutableHandlers
		tags map[any]struct{}
	}

	noBatch struct {
		Trampoline
	}

	batchHandler struct {
		Handler
		batch     *batch
		completed int32
	}

	noBatchHandler struct {
		Handler
	}

	// Batched wraps a Callback participating in a batch operation.
	Batched[T any] struct {
		Source   T
		Callback Callback
	}
)

// batch

func (b *batch) ShouldBatch(tag any) bool {
	if len(b.tags) == 0 {
		return true
	}
	_, ok := b.tags[tag]
	return ok
}

func (b *batch) Complete(
	composer Handler,
) *promise.Promise[[]any] {
	var results []*promise.Promise[any]
	for _, handler := range b.Handlers() {
		if c, ok := handler.(batching); ok {
			if r, pr, err := c.CompleteBatch(composer); err != nil {
				return promise.Reject[[]any](err)
			} else if pr == nil {
				results = append(results, promise.Resolve(r))
			} else {
				results = append(results, pr)
			}
		}
	}
	return promise.All(context.Background(), results...)
}

func (b *noBatch) CanBatch() bool {
	return false
}

// batchHandler

func (b *batchHandler) NoConstructor() {}

func (b *batchHandler) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, b)
	cb := callback
	if comp, ok := cb.(*Composition); ok {
		cb = comp.Callback()
	}
	switch cb := cb.(type) {
	case *Provides:
		if typ, ok := cb.key.(reflect.Type); ok {
			if typ == batchType {
				if batch := b.batch; batch != nil {
					return cb.ReceiveResult(batch, true, composer)
				}
			} else if typ.Implements(batchingType) {
				if batch := b.batch; batch != nil {
					for _, h := range batch.Handlers() {
						if _, ok := h.(batching); ok {
							return cb.ReceiveResult(h, true, composer)
						}
					}
					if batcher, err := newWithTag(typ, ""); err != nil {
						batch.AddHandlers(batcher)
						return cb.ReceiveResult(batcher, true, composer)
					}
				}
			}
		}
	default:
		if batch := b.batch; batch != nil {
			if check, ok := callback.(interface{
				CanBatch() bool
			}); !ok || check.CanBatch() {
				if r := batch.Handle(callback, greedy, composer);
					r.Handled() && !r.Stop() {
					return r
				}
			}
		}
	}
	return b.Handler.Handle(callback, greedy, composer)
}

func (b *batchHandler) Complete(
	promises ...*promise.Promise[any],
) *promise.Promise[[]any] {
	if !atomic.CompareAndSwapInt32(&b.completed, 0, 1) {
		panic("batch has already completed")
	}
	batch := b.batch
	b.batch = nil
	if results := batch.Complete(b); len(promises) == 0 {
		return results
	} else {
		ctx := context.Background()
		return promise.Then(results, ctx, func(res []any) []any {
			if _, err := promise.All(ctx, promises...).
				Await(context.Background()); err != nil {
				panic(err)
			}
			return res
		})
	}
}

func (b *noBatchHandler) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if callback == nil {
		return NotHandled
	}
	tryInitializeComposer(&composer, b)
	cb := callback
	if comp, ok := callback.(*Composition); ok {
		cb = comp.Callback()
	}
	if p, ok := cb.(*Provides); ok &&  p.Key() == batchType {
		return NotHandled
	}
	nb := &noBatch{}
	nb.callback = callback
	return b.Handler.Handle(nb, greedy, composer)
}

func Batch(
	handler   Handler,
	configure func(Handler),
	tags      ...any,
) *promise.Promise[[]any] {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if configure == nil {
		panic("configure cannot be nil")
	}
	batch := &batchHandler{handler, newBatch(tags...), 0}
	configure(batch)
	return batch.Complete()
}

func BatchAsync[T any](
	handler   Handler,
	configure func(Handler) *promise.Promise[T],
	tags      ...any,
) *promise.Promise[[]any] {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	if configure == nil {
		panic("configure cannot be nil")
	}
	batch := &batchHandler{handler, newBatch(tags...), 0}
	return batch.Complete(configure(batch).Then(context.Background(), func(data any) any {
		return data
	}))
}

func BatchTag[T any](
	handler   Handler,
	configure func(Handler),
) *promise.Promise[[]any] {
	return Batch(handler, configure, TypeOf[T]())
}

func BatchTagAsync[T any, E any](
	handler   Handler,
	configure func(Handler) *promise.Promise[T],
) *promise.Promise[[]any] {
	return BatchAsync(handler, configure, TypeOf[E]())
}

var NoBatch BuilderFunc = func(handler Handler) Handler {
	if IsNil(handler) {
		panic("handler cannot be nil")
	}
	return &noBatchHandler{handler}
}

func GetBatch[TB batching](handler Handler, tags ...any) TB {
	var tb TB
	if batch, _, err := Resolve[*batch](handler); err == nil && batch != nil {
		for _, tag := range tags {
			if !batch.ShouldBatch(tag) {
				break
			}
		}
		for _, handler := range batch.Handlers() {
			if batcher, ok := handler.(TB); ok {
				return batcher
			}
		}
		if batcher, err := newWithTag(TypeOf[TB](), ""); err == nil {
			batch.AddHandlers(batcher)
			return batcher.(TB)
		}
	}
	return tb
}

func newBatch(tags ...any) *batch {
	if len(tags) == 0 {
		return &batch{}
	}
	tagMap := make(map[any]struct{})
	for _, tag := range tags {
		tagMap[tag] = struct{}{}
	}
	return &batch{tags: tagMap}
}

var (
	batchType    = TypeOf[*batch]()
	batchingType = TypeOf[batching]()
)