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
		promises := c.promises
		if count := len(promises); count > 0 {
			if count == 1 {
				return nil, promises[0].Then(func(any) any {
					return c.ensureResult(many)
				})
			} else {
				return nil, promise.All(promises...).Then(func(any) any {
					return c.ensureResult(many)
				})
			}
		} else {
			c.ensureResult(many)
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
		idx := len(c.results)
		c.results  = append(c.results, result)
		c.promises = append(c.promises, pr.Then(func(res any) any {
			if accept != nil {
				accept(res, composer)
				res = nil
			}
			if l := len(c.results); l > idx {
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
		return c.addResults(result, composer)
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
			CopySliceIndirect(res.([]any), target)
			return *target
		})
	}
	return
}

func (c *CallbackBase) ensureResult(many bool) any {
	if c.result == nil {
		results := slices.Filter(c.results, func(res any) bool {
			return !IsNil(res)
		})
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
			if strict {
				c.AddResult(res, composer)
			} else {
				switch reflect.TypeOf(res).Kind() {
				case reflect.Slice, reflect.Array:
					c.addResults(res, composer)
				default:
					c.AddResult(res, composer)
				}
			}
			return nil
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
		c.addResults(result, composer)
	default:
		return c.AddResult(result, composer)
	}
	return Handled
}

func (c *CallbackBase) addResults(
	list     any,
	composer Handler,
) HandleResult {
	res := NotHandled
	v := reflect.ValueOf(list)
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i).Interface()
		if !IsNil(val) {
			if res = res.Or(c.AddResult(val, composer)); res.stop {
				break
			}
		}
	}
	return res
}
