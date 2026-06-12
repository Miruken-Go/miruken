package miruken

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Callback represents an intention.
	Callback interface {
		ConstraintSource
		Key() any
		Source() any
		Target() any
		TargetForWrite() any
		Policy() Policy
		ResultCount() int
		Result(many bool) (any, *promise.Promise[any])
		SetResult(result any)
		ReceiveResult(
			result any,
			strict bool,
			composer Handler,
		) HandleResult
	}

	// CallbackGuard detects and prevents circular Callback dispatch.
	CallbackGuard interface {
		CanDispatch(
			handler any,
			binding Binding,
		) (reset func(), approved bool)
	}

	// AcceptResultFunc accepts or rejects callback results.
	AcceptResultFunc func(
		result any,
		composer Handler,
	) HandleResult

	// AcceptPromiseResultFunc adjusts promise callback results.
	AcceptPromiseResultFunc func(
		pa *promise.Promise[any],
	) *promise.Promise[any]

	// CallbackBase is abstract Callback implementation.
	CallbackBase struct {
		mu            sync.Mutex
		async         atomic.Bool // true once a promise result is added
		result        any
		results       []any
		target        any
		written       bool
		promises      []*promise.Promise[any]
		accept        AcceptResultFunc
		acceptPromise AcceptPromiseResultFunc
		constraints   []Constraint
	}

	// CallbackBuilder builds common CallbackBase.
	CallbackBuilder struct {
		target      any
		constraints []Constraint
	}

	// customizeDispatch customizes Callback dispatch.
	customizeDispatch interface {
		Dispatch(
			handler any,
			greedy bool,
			composer Handler,
		) HandleResult
	}

	// suppressDispatch opts out of Callback dispatch.
	suppressDispatch interface {
		SuppressDispatch()
	}

	// marks a list of results to be expanded.
	expandResults []any
)

// CallbackBase

func (c *CallbackBase) Source() any {
	return nil
}

func (c *CallbackBase) Target() any {
	return c.target
}

func (c *CallbackBase) TargetForWrite() any {
	target := c.target
	if !internal.IsNil(target) {
		c.written = true
	}
	return target
}

func (c *CallbackBase) ResultCount() int {
	c.mu.Lock()
	n := len(c.results)
	c.mu.Unlock()
	return n
}

func (c *CallbackBase) Result(
	many bool,
) (any, *promise.Promise[any]) {
	if c.async.Load() {
		c.mu.Lock()
		result := c.result
		promises := c.promises
		c.mu.Unlock()
		if result == nil {
			switch len(promises) {
			case 0:
				result = c.ensureResult(many, false)
			case 1:
				return nil, promises[0].Then(func(any) any {
					return c.ensureResult(many, true)
				})
			default:
				return nil, promise.All(nil, promises...).
					Then(func(any) any {
						return c.ensureResult(many, true)
					})
			}
		}
		return result, nil
	} else if c.result == nil {
		c.ensureResult(many, false)
	}
	return c.result, nil
}

func (c *CallbackBase) SetResult(result any) {
	c.result = result
}

func (c *CallbackBase) SetAcceptResult(
	accept AcceptResultFunc,
) {
	c.accept = accept
}

func (c *CallbackBase) SetAcceptPromiseResult(
	accept AcceptPromiseResultFunc,
) {
	c.acceptPromise = accept
}

func (c *CallbackBase) AddResult(
	result   any,
	composer Handler,
) HandleResult {
	if internal.IsNil(result) {
		return NotHandled
	}
	accept := c.accept
	if pr, ok := result.(promise.Reflect); ok && !internal.IsNil(pr) {
		c.mu.Lock()
		idx := len(c.results)
		c.results = append(c.results, result)
		c.async.Store(true)
		c.mu.Unlock()
		p := pr.Then(func(res any) any {
			if accept != nil {
				c.mu.Lock()
				if l := len(c.results); l > idx {
					c.results[idx] = nil
				}
				c.mu.Unlock()
				if !internal.IsNil(res) {
					accept(res, composer)
				}
			} else {
				c.mu.Lock()
				if l := len(c.results); l > idx {
					c.results[idx] = res
				}
				c.mu.Unlock()
			}
			return nil
		})
		c.mu.Lock()
		c.promises = append(c.promises, p)
		c.result = nil
		c.mu.Unlock()
	} else if accept == nil {
		c.mu.Lock()
		c.results = append(c.results, result)
		c.result = nil
		c.mu.Unlock()
	} else {
		return accept(result, composer)
	}
	return Handled
}

func (c *CallbackBase) ReceiveResult(
	result   any,
	strict   bool,
	composer Handler,
) HandleResult {
	if internal.IsNil(result) {
		return NotHandled
	}
	if strict {
		return c.includeResult(result, true, composer)
	}
	if isSliceOrArray(result) {
		_, r := c.processResults(false, result, composer)
		return r
	}
	return c.includeResult(result, false, composer)
}

func (c *CallbackBase) Constraints() []Constraint {
	return c.constraints
}

func (c *CallbackBase) ensureResult(many, expand bool) any {
	// Check under lock first - fast path if already computed.
	c.mu.Lock()
	if c.result != nil {
		r := c.result
		c.mu.Unlock()
		return r
	}
	// Snapshot c.results under lock so we can release before
	// calling unwrapResult, which may call AwaitAny and block.
	// Holding the lock while blocking would deadlock because the
	// promise resolution callbacks also acquire c.mu.
	snapshot := make([]any, len(c.results))
	copy(snapshot, c.results)
	c.mu.Unlock()

	var results []any
	if expand {
		for _, res := range snapshot {
			if internal.IsNil(res) {
				continue
			}
			if exp, ok := res.(expandResults); ok {
				results = append(results, exp...)
			} else {
				results = append(results, res)
			}
		}
	} else {
		for _, res := range snapshot {
			if !internal.IsNil(res) {
				results = append(results, res)
			}
		}
	}

	// Compute final result outside the lock (unwrapResult may block).
	var computed any
	switch {
	case many:
		computed = unwrapResult(results)
	case len(results) > 0:
		computed = unwrapResult(results[0])
	}

	// Write result under lock; double-check in case another goroutine
	// raced through the same path and already set it.
	c.mu.Lock()
	if c.result == nil {
		c.result = computed
		if !(c.written || internal.IsNil(c.target)) {
			if many {
				internal.CopySliceIndirect(results, c.target)
			} else if computed != nil {
				internal.CopyIndirect(computed, c.target)
			}
			c.written = true
		}
	}
	r := c.result
	c.mu.Unlock()
	return r
}

func (c *CallbackBase) includeResult(
	result   any,
	strict   bool,
	composer Handler,
) HandleResult {
	if internal.IsNil(result) {
		return NotHandled
	}
	if pr, ok := result.(promise.Reflect); ok && !internal.IsNil(pr) {
		pp := pr.Then(func(res any) any {
			if !(strict || internal.IsNil(res)) {
				// Squash list into expando result
				if isSliceOrArray(res) {
					r, _ := c.processResults(true, res, composer)
					return r
				}
			}
			return res
		})
		if accept := c.acceptPromise; accept != nil {
			pp = accept(pp)
		}
		return c.AddResult(pp, composer)
	} else if strict {
		return c.AddResult(result, composer)
	}
	if isSliceOrArray(result) {
		c.processResults(false, result, composer)
		return Handled
	}
	return c.AddResult(result, composer)
}

// processResults adds an array or slice to the callbacks results.
// If squash is requested, the results are encoded in a special
// expandResults type that is expanded when results are requested.
// This is used to allow in-place replacement to avoid locking the results.
func (c *CallbackBase) processResults(
	squash   bool,
	results  any,
	composer Handler,
) (expandResults, HandleResult) {
	res := NotHandled
	var expand expandResults
	addItem := func(val any) bool {
		if !internal.IsNil(val) {
			if squash {
				expand = append(expand, val)
			} else if res = res.Or(c.AddResult(val, composer)); res.stop {
				return false
			}
		}
		return true
	}
	// Fast paths for []any and expandResults (most common cases).
	// expandResults is a named type over []any so it doesn't match
	// the []any type assertion and would otherwise fall through to
	// the reflection path unnecessarily.
	switch s := results.(type) {
	case []any:
		for _, val := range s {
			if !addItem(val) {
				break
			}
		}
	case expandResults:
		for _, val := range ([]any)(s) {
			if !addItem(val) {
				break
			}
		}
	default:
		// Fallback to reflection for typed slices from external handlers.
		// For pointer element types this is cheap; value types incur one
		// heap allocation per element (unavoidable without unsafe).
		v := reflect.ValueOf(s)
		for i := range v.Len() {
			if !addItem(v.Index(i).Interface()) {
				break
			}
		}
	}
	return expand, res
}

// CallbackBuilder

func (b *CallbackBuilder) IntoTarget(
	target any,
) *CallbackBuilder {
	if internal.IsNil(target) {
		panic("target cannot be nil")
	}
	b.target = target
	return b
}

func (b *CallbackBuilder) WithConstraints(
	constraints ...any,
) *CallbackBuilder {
	for _, constraint := range constraints {
		switch c := constraint.(type) {
		case nil:
		case string:
			b.constraints = append(b.constraints, new(Named(c)))
		case Constraint:
			b.constraints = append(b.constraints, c)
		case map[any]any:
			var m Metadata = c
			b.constraints = append(b.constraints, &m)
		case ConstraintSource:
			b.constraints = append(b.constraints, c.Constraints()...)
		default:
			panic(fmt.Sprintf("unrecognized constraint: %T", constraint))
		}
	}
	return b
}

func (b *CallbackBuilder) CallbackBase() CallbackBase {
	return CallbackBase{target: b.target, constraints: b.constraints}
}

// unwrapResult unwraps the result if it's a promise.
// During processing of a callback, it may be  promoted
// to an asynchronous operation, so it must be unwrapped.
//
//	e.g.  async filter, async args
// isSliceOrArray checks if val is a slice or array type,
// using fast-path type switches before falling back to reflection.
func isSliceOrArray(val any) bool {
	switch val.(type) {
	case []any, expandResults:
		return true
	}
	k := reflect.TypeOf(val).Kind()
	return k == reflect.Slice || k == reflect.Array
}

func unwrapResult(result any) any {
	if result == nil {
		return nil
	}
	if pr, ok := result.(promise.Reflect); ok && !internal.IsNil(pr) {
		if r, err := pr.AwaitAny(); err != nil {
			panic(err)
		} else {
			result = r
		}
	}
	return result
}
