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
			s   SideEffect,
			ctx HandleContext,
		) ([]any, *promise.Promise[[]any], error)
	}

	// LateSideEffect provides a late binding adapter
	// for SideEffect's with custom dependencies.
	LateSideEffect struct {}

	applyBinding struct {
		method reflect.Method
		ctxIdx int
		args   []arg
	}
)


func (l LateSideEffect) Apply(
	sideEffect SideEffect,
	ctx        HandleContext,
)  ([]any, *promise.Promise[[]any], error) {
	return lateApply(sideEffect, ctx)
}


func lateApply(
	sideEffect SideEffect,
	ctx        HandleContext,
)  (out []any, po *promise.Promise[[]any], err error) {
	var binding *applyBinding
	if binding, err = getLateApply(sideEffect); err != nil {
		return
	}
	if out, po, err = binding.invoke(sideEffect, ctx); err != nil {
		return
	} else if po == nil {
		po,  _ = out[1].(*promise.Promise[[]any])
		err, _ = out[2].(error)
		out, _ = out[0].([]any)
	} else {
		po = promise.Then(po, func(o []any) []any {
			if err, ok := o[2].(error); ok {
				panic(err)
			} else if ro, ok := o[0].([]any); ok {
				return ro
			} else {
				return nil
			}
		})
	}
	return
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
				lateApplyType.NumIn() < 1 || lateApplyType.NumOut() < 3 {
				goto Invalid
			} else if lateApplyType.Out(0) != anySliceType ||
				lateApplyType.Out(1) != promiseAnySliceType ||
				lateApplyType.Out(2) != errorType {
				goto Invalid
			} else {
				skip    := 1 // skip receiver
				numArgs := lateApplyType.NumIn()
				binding = &applyBinding{method: lateApply}
				for i := 1; i < 2 && i < numArgs; i++ {
					if lateApplyType.In(i) == handleCtxType {
						if binding.ctxIdx > 0 {
							return nil, &MethodBindingError{lateApply,
								fmt.Errorf("duplicate HandleContext arg at index %v and %v", binding.ctxIdx, i)}
						}
						binding.ctxIdx = i
						skip++
					}
				}
				args := make([]arg, numArgs-skip)
				if err := buildDependencies(lateApplyType, skip, numArgs, args, 0); err != nil {
					err = fmt.Errorf("LateApply: %w", err)
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
	return nil, fmt.Errorf(`side-effect: %v has no matching "LateApply" method`, typ)
}

func (a *applyBinding) invoke(
	s   SideEffect,
	ctx HandleContext,
) ([]any, *promise.Promise[[]any], error) {
	initArgs := []any{s}
	if a.ctxIdx == 1 {
		initArgs = append(initArgs, ctx)
	}
	return callFunc(a.method.Func, ctx, a.args, initArgs...)
}


var (
	lateApplyLock sync.RWMutex
	lateApplyMap = make(map[reflect.Type]*applyBinding)
)