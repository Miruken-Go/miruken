package miruken

import (
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/slices"
	"reflect"
)

type (
	// Callback represents any intention.
	Callback interface {
		Key() any
		Source() any
		Policy() Policy
		ResultCount() int
		Result(many bool) (any, *promise.Promise[any])
		SetResult(result any)
		ReceiveResult(
			result   any,
			strict   bool,
			composer Handler,
		) HandleResult
	}

	// CallbackGuard detects and prevents circular Callback dispatch.
	CallbackGuard interface {
		CanDispatch(
			handler any,
			binding Binding,
		) (reset func (), approved bool)
	}

	// AcceptResultFunc accepts or rejects callback results.
	AcceptResultFunc func(
		result   any,
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
		promises      []*promise.Promise[any]
		accept        AcceptResultFunc
		acceptPromise AcceptPromiseResultFunc
	}

	// customizeDispatch customizes Callback dispatch.
	customizeDispatch interface {
		Dispatch(
			handler  any,
			greedy   bool,
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

func (c *CallbackBase) Source() any {
	return nil
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
			return nil, promise.All(c.promises...).Then(func(any) any {
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
	if IsNil(result) {
		return NotHandled
	}
	accept := c.accept
	if pr, ok := result.(promise.Reflect); ok {
		// To avoid locking the results, promises are added to
		// the results and promises list.  When resolved, the
		// result is replaced at the same position.  A special
		// expandResults type is used when the promise resolves
		// in a list of results.
		idx := len(c.results)
		c.results  = append(c.results, result)
		c.promises = append(c.promises, pr.Then(func(res any) any {
			if accept != nil  {
				if l := len(c.results); l > idx {
					c.results[idx] = nil
				}
				if !IsNil(res) {
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
	if IsNil(result) {
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

func CoerceResult[T any](
	callback Callback,
	target   *T,
) (t T, tp *promise.Promise[T], _ error) {
	if target == nil {
		target = &t
	}
	if result, p := callback.Result(false); p == nil {
		CopyIndirect(result, target)
	} else {
		tp = promise.Then(p, func(res any) T {
			// During processing of the callback, it may be
			// promoted to asynchronous operation.
			//   e.g.  async filter, async args
			// It is necessary to unwrap the promise to obtain
			// the correct result to bind to.
			if !reflect.TypeOf(res).AssignableTo(TypeOf[T]()) {
				if pr, ok := res.(promise.Reflect); ok {
					if r, err := pr.AwaitAny(); err != nil {
						panic(err)
					} else {
						res = r
					}
				}
			}
			CopyIndirect(res, target)
			return *target
		})
	}
	return
}

func CoerceResults[T any](
	callback Callback,
	target   *[]T,
) (t []T, tp *promise.Promise[[]T], _ error) {
	if target == nil {
		target = &t
	}
	if result, p := callback.Result(true); p == nil {
		CopySliceIndirect(result.([]any), target)
	} else {
		tp = promise.Then(p, func(res any) []T {
			if !reflect.TypeOf(res).AssignableTo(TypeOf[T]()) {
				if pr, ok := res.(promise.Reflect); ok {
					if r, err := pr.AwaitAny(); err != nil {
						panic(err)
					} else {
						res = r
					}
				}
			}
			CopySliceIndirect(res.([]any), target)
			return *target
		})
	}
	return
}

func (c *CallbackBase) ensureResult(many bool, expand bool) any {
	if c.result == nil {
		var results []any
		if expand {
			results = slices.FlatMap[any, any](c.results, func(res any) []any {
				if IsNil(res) {
					return nil
				}
				if expand, ok := res.(expandResults); ok {
					return expand
				}
				return []any{res}
			})
		} else {
			results = slices.Filter(c.results, func(res any) bool {
				return !IsNil(res)
			})
		}
		if many {
			c.result = results
		} else if len(results) == 0 {
			c.result = nil
		} else {
			c.result = results[0]
		}
	}
	return c.result
}

func (c *CallbackBase) includeResult(
	result   any,
	strict   bool,
	composer Handler,
) HandleResult {
	if IsNil(result) {
		return NotHandled
	}
	if pr, ok := result.(promise.Reflect); ok {
		pp := pr.Then(func(res any) any {
			if !strict {
				// Squash list into expando result
				switch reflect.TypeOf(res).Kind() {
				case reflect.Slice, reflect.Array:
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
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i).Interface()
		if !IsNil(val) {
			if squash {
				expand = append(expand, val)
			} else if res = res.Or(c.AddResult(val, composer)); res.stop {
				break
			}
		}
	}
	return expand, res
}
