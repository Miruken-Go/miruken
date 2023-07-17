package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sync"
)

type (
	// SideEffect encapsulates a custom behavior to be
	// executed when returned from a Handler.
	// These behaviors generally represent interactions
	// with external entities i.e. databases and other IO.
	SideEffect interface {
		Apply(
			// self provided to facilitate late bindings
			self SideEffect,
			ctx  HandleContext,
		) (promise.Reflect, error)
	}

	// SideEffectAdapter is an adapter for implementing a
	// SideEffect using late binding method resolution.
	SideEffectAdapter struct {}

	// sideEffectBinding describes the method used by a
	// SideEffectAdapter to apply the SideEffect.
	sideEffectBinding struct {
		method reflect.Method
		ctxIdx int
		refIdx int
		errIdx int
		args   []arg
	}
)


func (l SideEffectAdapter) Apply(
	self SideEffect,
	ctx  HandleContext,
)  (promise.Reflect, error) {
	if binding, err := getLateApply(self); err != nil {
		return nil, err
	} else {
		return binding.invoke(self, ctx)
	}
}


func getLateApply(
	sideEffect SideEffect,
) (*sideEffectBinding, error) {
	sideEffectBindingLock.RLock()
	typ := reflect.TypeOf(sideEffect)
	binding := sideEffectBindingMap[typ]
	sideEffectBindingLock.RUnlock()
	if binding == nil {
		sideEffectBindingLock.Lock()
		defer sideEffectBindingLock.Unlock()
		if binding = sideEffectBindingMap[typ]; binding == nil {
			if lateApply, ok := typ.MethodByName("LateApply"); !ok {
				goto Invalid
			} else if lateApplyType := lateApply.Type;
				lateApplyType.NumIn() < 1 || lateApplyType.NumOut() > 2 {
				goto Invalid
			} else {
				// Output can be promise, error or both with error last
				refIdx, errIdx := -1, -1
				numOut := lateApplyType.NumOut()
				for i := 0; i < numOut; i++ {
					out := lateApplyType.Out(i)
					if out.AssignableTo(promiseReflectType) {
						if i != 0 {
							goto Invalid
						}
						refIdx = i
					} else if out.AssignableTo(errorType) {
						if i != numOut-1 {
							goto Invalid
						}
						errIdx = i
					} else {
						goto Invalid
					}
				}
				skip    := 1 // skip receiver
				numArgs := lateApplyType.NumIn()
				binding = &sideEffectBinding{method: lateApply, refIdx: refIdx, errIdx: errIdx}
				for i := 1; i < 2 && i < numArgs; i++ {
					if lateApplyType.In(i) == handleCtxType {
						if binding.ctxIdx > 0 {
							return nil, &MethodBindingError{lateApply,
								fmt.Errorf("side-effect: %v duplicate HandleContext arg at index %v and %v",
									typ, binding.ctxIdx, i)}
						}
						binding.ctxIdx = i
						skip++
					}
				}
				args := make([]arg, numArgs-skip)
				if err := buildDependencies(lateApplyType, skip, numArgs, args, 0); err != nil {
					err = fmt.Errorf("side-effect: %v \"LateApply\": %w", typ, err)
					return nil, &MethodBindingError{lateApply, err}
				}
				binding.args = args
				sideEffectBindingMap[typ] = binding
			}
		}
	}
	if binding != nil {
		return binding, nil
	}
Invalid:
	return nil, fmt.Errorf(`side-effect: %v has no valid "LateApply" method`, typ)
}

func (a *sideEffectBinding) invoke(
	s   SideEffect,
	ctx HandleContext,
) (promise.Reflect, error) {
	initArgs := []any{s}
	if a.ctxIdx == 1 {
		initArgs = append(initArgs, ctx)
	}
	fun := a.method.Func
	fromIndex := len(initArgs)
	ra, pa, err := resolveFuncArgs(fun, a.args, fromIndex, ctx)
	if err != nil {
		return nil, err
	} else if pa == nil {
		out := callFuncWithArgs(fun, ra, initArgs)
		if errIdx := a.errIdx; errIdx >= 0 {
			if oe, ok := out[errIdx].(error); ok && oe != nil {
				return nil, oe
			}
		}
		if refIdx := a.refIdx; refIdx >= 0 {
			if po, ok := out[refIdx].(promise.Reflect); ok && !IsNil(po) {
				return po, nil
			}
		}
		return nil, nil
	} else {
		return promise.Then(pa, func(ra []reflect.Value) any {
			out := callFuncWithArgs(fun, ra, initArgs)
			if errIdx := a.errIdx; errIdx >= 0 {
				if oe, ok := out[errIdx].(error); ok && oe != nil {
					panic(oe)
				}
			}
			if refIdx := a.refIdx; refIdx >= 0 {
				if po, ok := out[refIdx].(promise.Reflect); ok && !IsNil(po) {
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
}


var (
	sideEffectBindingLock sync.RWMutex
	sideEffectBindingMap = make(map[reflect.Type]*sideEffectBinding)
	promiseReflectType   = TypeOf[promise.Reflect]()
	sideEffectType       = TypeOf[SideEffect]()
)