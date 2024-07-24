package miruken

import (
	"fmt"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Effect describes an action to be performed.
	// Effects are additional outputs from a Handler.
	// These actions generally represent interactions
	// with external entities i.e. databases and other IO.
	Effect interface {
		Apply(HandleContext) (promise.Reflect, error)
	}

	// CascadeEffect is a standard Effect for cascading callbacks.
	CascadeEffect struct {
		callbacks   []any
		constraints []any
		handler     Handler
		greedy      bool
	}

	// effectAdapter is an adapter for implementing an
	// Effect using late binding method resolution.
	effectAdapter struct{
		effect  any
		binding *effectBinding
	}

	// effectBinding describes the method used by a
	// effectAdapter to apply the Effect dynamically.
	effectBinding struct {
		funcCall
		ctxIdx int
		refIdx int
		errIdx int
	}
)


// CascadeEffect

func (c *CascadeEffect) WithConstraints(
	constraints ...any,
) *CascadeEffect {
	c.constraints = constraints
	return c
}

func (c *CascadeEffect) WithHandler(
	handler Handler,
) *CascadeEffect {
	c.handler = handler
	return c
}

func (c *CascadeEffect) Greedy(
	greedy bool,
) *CascadeEffect {
	c.greedy = greedy
	return c
}

func (c *CascadeEffect) Apply(
	ctx HandleContext,
) (promise.Reflect, error) {
	callbacks := c.callbacks
	if len(callbacks) == 0 {
		return nil, nil
	}

	handler := c.handler
	if internal.IsNil(handler) {
		handler = ctx.Composer
	}

	var promises []*promise.Promise[any]

	for _, callback := range callbacks {
		var pc *promise.Promise[any]
		var err error
		if c.greedy {
			pc, err = CommandAll(handler, callback, c.constraints...)
		} else {
			pc, err = Command(handler, callback, c.constraints...)
		}
		if err != nil {
			return nil, err
		} else if pc != nil {
			promises = append(promises, pc)
		}
	}

	switch len(promises) {
	case 0:
		return nil, nil
	case 1:
		return promises[0], nil
	default:
		return promise.All(nil, promises...), nil
	}
}

// Cascade is a fluent builder to cascade callbacks.
func Cascade(callbacks ...any) *CascadeEffect {
	return &CascadeEffect{callbacks: callbacks}
}

// MakeEffect creates an Effect from anything.
// If the argument is already an Effect it is returned.
// If require is true, an error is returned if the argument
// cannot be coerced into an Effect.
func MakeEffect(
	effect  any,
	require bool,
) (Effect, error) {
	if internal.IsNil(effect) {
		return nil, nil
	}
	if i, ok := effect.(Effect); ok {
		return i, nil
	}
	typ := reflect.TypeOf(effect)
	if binding, err := getEffectMethod(typ, require); err != nil {
		return nil, err
	} else if binding != nil {
		return &effectAdapter{effect, binding}, nil
	}
	return nil, nil
}

// MakeEffects creates Effect from anything.
// If an argument is already an Effect it is returned.
// Unrecognized effects are returned as is.
// If require is true, an error is returned if any argument
// cannot be coerced into an Effect.
func MakeEffects(
	require bool,
	effects []any,
) ([]Effect, []any, error) {
	var xs []any
	ins := make([]Effect, 0, len(effects))
	for _, candidate := range effects {
		if effect, err := MakeEffect(candidate, require); err != nil {
			return nil, nil, err
		} else if effect != nil {
			ins = append(ins, effect)
		} else if !internal.IsNil(candidate) {
			xs = append(xs, candidate)
		}
	}
	return ins, xs, nil
}

// ValidEffect returns true if the argument is an Effect or
// can be converted to an Effect.
func ValidEffect(
	typ reflect.Type,
) (bool, error) {
	if internal.IsNil(typ) {
		return false, nil
	}
	if typ.AssignableTo(effectType) {
		return true, nil
	}
	binding, err := getEffectMethod(typ, false)
	if err != nil {
		return false, err
	}
	return binding != nil, nil
}


func (i *effectAdapter) Apply(
	ctx HandleContext,
) (promise.Reflect, error) {
	return i.binding.invoke(i.effect, ctx)
}

// getEffectMethod discovers a suitable dynamic Effect method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getEffectMethod(
	typ     reflect.Type,
	require bool,
) (*effectBinding, error) {
	if bindings := effectBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
	}
	effectBindingLock.Lock()
	defer effectBindingLock.Unlock()
	bindings := effectBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
		sb := maps.Clone(*bindings)
		bindings = &sb
	} else {
		bindings = &map[reflect.Type]effectBinding{}
	}
	for i := range typ.NumMethod() {
		method := typ.Method(i)
		if method.Name != "Apply" {
			continue
		}
		if lateApplyType := method.Type; lateApplyType.NumIn() < 1 || lateApplyType.NumOut() > 2 {
			break
		} else {
			// Output can be promise, error or both with error last
			refIdx, errIdx := -1, -1
			numOut := lateApplyType.NumOut()
			for i := range numOut {
				out := lateApplyType.Out(i)
				if out.AssignableTo(promiseReflectType) {
					if i != 0 {
						continue
					}
					refIdx = i
				} else if out.AssignableTo(internal.ErrorType) {
					if i != numOut-1 {
						continue
					}
					errIdx = i
				} else {
					continue
				}
			}
			skip    := 1 // skip receiver
			numArgs := lateApplyType.NumIn()
			binding := effectBinding{refIdx: refIdx, errIdx: errIdx}
			for i := 1; i < 2 && i < numArgs; i++ {
				if lateApplyType.In(i) == handleCtxType {
					if binding.ctxIdx > 0 {
						return nil, &MethodBindingError{&method,
							fmt.Errorf("effect: %v duplicate HandleContext arg at index %v and %v",
								typ, binding.ctxIdx, i)}
					}
					binding.ctxIdx = i
					skip++
				}
			}
			args := make([]arg, numArgs-skip)
			if err := buildDependencies(lateApplyType, skip, numArgs, args, 0); err != nil {
				err = fmt.Errorf("effect: %v %q: %w", typ, method.Name, err)
				return nil, &MethodBindingError{&method, err}
			}
			binding.funcCall.fun = method.Func
			binding.funcCall.args = args
			(*bindings)[typ] = binding
			effectBindingMap.Store(bindings)
			return &binding, nil
		}
	}
	if require {
		return nil, fmt.Errorf(`effect: %v has no compatible "Apply" method`, typ)
	}
	return nil, nil
}

func (b effectBinding) invoke(
	effect any,
	ctx    HandleContext,
) (promise.Reflect, error) {
	initArgs := []any{effect}
	if b.ctxIdx == 1 {
		initArgs = append(initArgs, ctx)
	}
	out, pout, err := b.Invoke(ctx, initArgs...)
	if err != nil {
		return nil, err
	} else if pout == nil {
		if errIdx := b.errIdx; errIdx >= 0 {
			if oe, ok := out[errIdx].(error); ok && oe != nil {
				return nil, oe
			}
		}
		if refIdx := b.refIdx; refIdx >= 0 {
			if po, ok := out[refIdx].(promise.Reflect); ok && !internal.IsNil(po) {
				return po, nil
			}
		}
		return nil, nil
	}
	return promise.Then(pout, func(out []any) any {
		if errIdx := b.errIdx; errIdx >= 0 {
			if oe, ok := out[errIdx].(error); ok && oe != nil {
				panic(oe)
			}
		}
		if refIdx := b.refIdx; refIdx >= 0 {
			if po, ok := out[refIdx].(promise.Reflect); ok && !internal.IsNil(po) {
				if oa, oe := po.AwaitAny(); oe != nil {
					panic(oe)
				} else {
					return oa
				}
			}
		}
		return nil
	}), nil
}


var (
	effectBindingLock sync.Mutex
	effectBindingMap   = atomic.Pointer[map[reflect.Type]effectBinding]{}
	effectType         = reflect.TypeFor[Effect]()
	promiseReflectType = reflect.TypeFor[promise.Reflect]()
)
