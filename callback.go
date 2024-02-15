package miruken

import (
	"fmt"
	"reflect"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/internal/slices"
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
	return len(c.results)
}

func (c *CallbackBase) Result(
	many bool,
) (any, *promise.Promise[any]) {
	if c.result == nil {
		switch len(c.promises) {
		case 0:
			c.ensureResult(many, false)
		case 1:
			return nil, c.promises[0].Then(func(any) any {
				return c.ensureResult(many, true)
			})
		default:
			return nil, promise.All(nil, c.promises...).
				Then(func(any) any {
					return c.ensureResult(many, true)
				})
		}
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
		// To avoid locking the results, promises are added to
		// the results and promises list.  When resolved, the
		// result is replaced at the same position.  A special
		// expandResults type is used when the promise Resolves
		// in a list of results.
		idx := len(c.results)
		c.results = append(c.results, result)
		c.promises = append(c.promises, pr.Then(func(res any) any {
			if accept != nil {
				if l := len(c.results); l > idx {
					c.results[idx] = nil
				}
				if !internal.IsNil(res) {
					accept(res, composer)
				}
			} else if l := len(c.results); l > idx {
				c.results[idx] = res
			}
			return nil
		}))
	} else if accept == nil {
		c.results = append(c.results, result)
	} else {
		return accept(result, composer)
	}
	c.result = nil
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
	switch reflect.TypeOf(result).Kind() {
	case reflect.Slice, reflect.Array:
		_, r := c.processResults(false, result, composer)
		return r
	default:
		return c.includeResult(result, false, composer)
	}
}

func (c *CallbackBase) Constraints() []Constraint {
	return c.constraints
}

func (c *CallbackBase) ensureResult(many, expand bool) any {
	if c.result == nil {
		var results []any
		if expand {
			results = slices.FlatMap[any, any](c.results, func(res any) []any {
				if internal.IsNil(res) {
					return nil
				}
				if expand, ok := res.(expandResults); ok {
					return expand
				}
				return []any{res}
			})
		} else {
			results = slices.Filter(c.results, func(res any) bool {
				return !internal.IsNil(res)
			})
		}
		switch {
		case many:
			c.result = unwrapResult(results)
			if !(c.written || internal.IsNil(c.target)) {
				internal.CopySliceIndirect(results, c.target)
				c.written = true
			}
		case len(results) == 0:
			c.result = nil
		default:
			c.result = unwrapResult(results[0])
			if !(c.written || internal.IsNil(c.target)) {
				internal.CopyIndirect(c.result, c.target)
				c.written = true
			}
		}
	}
	return c.result
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
				switch reflect.TypeOf(res).Kind() {
				case reflect.Slice, reflect.Array:
					r, _ := c.processResults(true, res, composer)
					return r
				default:
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
	switch reflect.TypeOf(result).Kind() {
	case reflect.Slice, reflect.Array:
		c.processResults(false, result, composer)
	default:
		return c.AddResult(result, composer)
	}
	return Handled
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
	v := reflect.ValueOf(results)
	for i := range v.Len() {
		val := v.Index(i).Interface()
		if !internal.IsNil(val) {
			if squash {
				expand = append(expand, val)
			} else if res = res.Or(c.AddResult(val, composer)); res.stop {
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
			n := Named(c)
			b.constraints = append(b.constraints, &n)
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
