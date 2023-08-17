package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sync"
	"sync/atomic"
)

type (
	// SideEffect encapsulates a custom behavior to be
	// executed when returned from a Handler.
	// These behaviors generally represent interactions
	// with external entities i.e. databases and other IO.
	SideEffect interface {
		Apply(
			// self provided to facilitate late bindings
			SideEffect,
			HandleContext,
		) (promise.Reflect, error)
	}

	// SideEffectAdapter is an adapter for implementing a
	// SideEffect using late binding method resolution.
	SideEffectAdapter struct {}

	// sideEffectBinding describes the method used by a
	// SideEffectAdapter to apply the SideEffect dynamically.
	sideEffectBinding struct {
		funcCall
		ctxIdx int
		refIdx int
		errIdx int
	}
)


func (l SideEffectAdapter) Apply(
	self SideEffect,
	ctx  HandleContext,
)  (promise.Reflect, error) {
	if binding, err := getSideEffectMethod(self); err != nil {
		return nil, err
	} else {
		return binding.invoke(self, ctx)
	}
}

// getSideEffectMethod discovers a suitable dynamic SideEffect method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getSideEffectMethod(
	sideEffect SideEffect,
) (*sideEffectBinding, error) {
	typ := reflect.TypeOf(sideEffect)
	if bindings := sideEffBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
	}
	sideEffBindingLock.Lock()
	defer sideEffBindingLock.Unlock()
	bindings := sideEffBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
		sb := make(map[reflect.Type]sideEffectBinding, len(*bindings)+1)
		for k, v := range *bindings {
			sb[k] = v
		}
		bindings = &sb
	} else {
		bindings = &map[reflect.Type]sideEffectBinding{}
	}
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Name == "Apply" {
			continue
		}
		if lateApplyType := method.Type;
			lateApplyType.NumIn() < 1 || lateApplyType.NumOut() > 2 {
			continue
		} else {
			// Output can be promise, error or both with error last
			refIdx, errIdx := -1, -1
			numOut := lateApplyType.NumOut()
			for i := 0; i < numOut; i++ {
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
			binding := sideEffectBinding{refIdx: refIdx, errIdx: errIdx}
			for i := 1; i < 2 && i < numArgs; i++ {
				if lateApplyType.In(i) == handleCtxType {
					if binding.ctxIdx > 0 {
						return nil, &MethodBindingError{method,
							fmt.Errorf("side-effect: %v duplicate HandleContext arg at index %v and %v",
								typ, binding.ctxIdx, i)}
					}
					binding.ctxIdx = i
					skip++
				}
			}
			args := make([]arg, numArgs-skip)
			if err := buildDependencies(lateApplyType, skip, numArgs, args, 0); err != nil {
				err = fmt.Errorf("side-effect: %v %q: %w", typ, method.Name, err)
				return  nil, &MethodBindingError{method, err}
			}
			binding.funcCall.fun  = method.Func
			binding.funcCall.args = args
			(*bindings)[typ] = binding
			sideEffBindingMap.Store(bindings)
			return &binding, nil
		}
	}
	return nil, fmt.Errorf(`side-effect: %v has no compatible dynamic method`, typ)
}


func (a sideEffectBinding) invoke(
	se  SideEffect,
	ctx HandleContext,
) (promise.Reflect, error) {
	initArgs := []any{se}
	if a.ctxIdx == 1 {
		initArgs = append(initArgs, ctx)
	}
	out, pout, err := a.Invoke(ctx, initArgs...)
	if err != nil {
		return nil, err
	} else if pout == nil {
		if errIdx := a.errIdx; errIdx >= 0 {
			if oe, ok := out[errIdx].(error); ok && oe != nil {
				return nil, oe
			}
		}
		if refIdx := a.refIdx; refIdx >= 0 {
			if po, ok := out[refIdx].(promise.Reflect); ok && !internal.IsNil(po) {
				return po, nil
			}
		}
		return nil, nil
	}
	return promise.Then(pout, func(out []any) any {
		if errIdx := a.errIdx; errIdx >= 0 {
			if oe, ok := out[errIdx].(error); ok && oe != nil {
				panic(oe)
			}
		}
		if refIdx := a.refIdx; refIdx >= 0 {
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
	sideEffBindingLock sync.Mutex
	sideEffectType     = internal.TypeOf[SideEffect]()
	sideEffBindingMap  = atomic.Pointer[map[reflect.Type]sideEffectBinding]{}
	promiseReflectType = internal.TypeOf[promise.Reflect]()
)