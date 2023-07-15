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
	// with external entities i.e. databases and files
	SideEffect interface {
		Apply(
			// self provided to facilitate late bindings
			self SideEffect,
			ctx  HandleContext,
		) (promise.Reflect, error)
	}

	// LateSideEffect provides a late binding adapter
	// for SideEffect's with custom dependencies.
	LateSideEffect struct {}

	applyBinding struct {
		method reflect.Method
		ctxIdx int
		prIdx  int
		errIdx int
		args   []arg
	}
)


func (l LateSideEffect) Apply(
	self SideEffect,
	ctx  HandleContext,
)  (promise.Reflect, error) {
	return lateApply(self, ctx)
}


func lateApply(
	sideEffect SideEffect,
	ctx        HandleContext,
)  (p promise.Reflect, err error) {
	var binding *applyBinding
	if binding, err = getLateApply(sideEffect); err != nil {
		return
	}
	return binding.invoke(sideEffect, ctx)
}


func getLateApply(
	sideEffect SideEffect,
) (*applyBinding, error) {
	lateApplyLock.RLock()
	typ := reflect.TypeOf(sideEffect)
	binding := lateApplyMap[typ]
	lateApplyLock.RUnlock()
	if binding == nil {
		lateApplyLock.Lock()
		defer lateApplyLock.Unlock()
		if binding = lateApplyMap[typ]; binding == nil {
			if lateApply, ok := typ.MethodByName("LateApply"); !ok {
				goto Invalid
			} else if lateApplyType := lateApply.Type;
				lateApplyType.NumIn() < 1 || lateApplyType.NumOut() > 2 {
				goto Invalid
			} else {
				// Output can be promise, error or both with error last
				prIdx, errIdx := -1, -1
				numOut := lateApplyType.NumOut()
				for i := 0; i < numOut; i++ {
					out := lateApplyType.Out(i)
					if out.AssignableTo(promiseReflectType) {
						if i != 0 {
							goto Invalid
						}
						prIdx = i
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
				binding = &applyBinding{method: lateApply, prIdx: prIdx, errIdx: errIdx}
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
				lateApplyMap[typ] = binding
			}
		}
	}
	if binding != nil {
		return binding, nil
	}
Invalid:
	return nil, fmt.Errorf(`side-effect: %v has no valid "LateApply" method`, typ)
}

func (a *applyBinding) invoke(
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
		if prIdx := a.prIdx; prIdx >= 0 {
			if po, ok := out[prIdx].(promise.Reflect); ok && !IsNil(po) {
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
			if prIdx := a.prIdx; prIdx >= 0 {
				if po, ok := out[prIdx].(promise.Reflect); ok && !IsNil(po) {
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
	lateApplyLock sync.RWMutex
	lateApplyMap       = make(map[reflect.Type]*applyBinding)
	promiseReflectType = TypeOf[promise.Reflect]()
	sideEffectType     = TypeOf[SideEffect]()
)