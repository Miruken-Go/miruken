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
	// Intent describes an action to be performed.
	// Intents are additional outputs from a Handler.
	// These actions generally represent interactions
	// with external entities i.e. databases and other IO.
	Intent interface {
		Apply(HandleContext) (promise.Reflect, error)
	}

	// intentAdapter is an adapter for implementing an
	// Intent using late binding method resolution.
	intentAdapter struct{
		intent  any
		binding *intentBinding
	}

	// intentBinding describes the method used by a
	// intentAdapter to apply the Intent dynamically.
	intentBinding struct {
		funcCall
		ctxIdx int
		refIdx int
		errIdx int
	}
)


// MakeIntent creates an Intent from anything.
// If the argument is already an Intent it is returned.
// If require is true, an error is returned if the argument
// cannot be coerced into an Intent.
func MakeIntent(intent any, require bool) (Intent, error) {
	if internal.IsNil(intent) {
		return nil, nil
	}
	if i, ok := intent.(Intent); ok {
		return i, nil
	}
	typ := reflect.TypeOf(intent)
	if binding, err := getIntentMethod(typ, require); err != nil {
		return nil, err
	} else if binding != nil {
		return &intentAdapter{intent, binding}, nil
	}
	return nil, nil
}

// MakeIntents creates Intent's from anything.
// If an argument is already an Intent it is returned.
// If require is true, an error is returned if any argument
// cannot be coerced into an Intent.
func MakeIntents(require bool, intents []any) ([]Intent, error) {
	ins := make([]Intent, 0, len(intents))
	for _, intent := range intents {
		if i, err := MakeIntent(intent, require); err != nil {
			return nil, err
		} else if i != nil {
			ins = append(ins, i)
		}
	}
	return ins, nil
}

// ValidIntent returns true if the argument is an Intent or
// can be converted to an Intent.
func ValidIntent(typ reflect.Type) (bool, error) {
	if internal.IsNil(typ) {
		return false, nil
	}
	if typ.AssignableTo(intentType) {
		return true, nil
	}
	binding, err := getIntentMethod(typ, false)
	if err != nil {
		return false, err
	}
	return binding != nil, nil
}


func (i *intentAdapter) Apply(
	ctx HandleContext,
) (promise.Reflect, error) {
	return i.binding.invoke(i.intent, ctx)
}

// getIntentMethod discovers a suitable dynamic Intent method.
// Uses the copy-on-write idiom since reads should be more frequent than writes.
func getIntentMethod(
	typ     reflect.Type,
	require bool,
) (*intentBinding, error) {
	if bindings := intentBindingMap.Load(); bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
	}
	intentBindingLock.Lock()
	defer intentBindingLock.Unlock()
	bindings := intentBindingMap.Load()
	if bindings != nil {
		if binding, ok := (*bindings)[typ]; ok {
			return &binding, nil
		}
		sb := maps.Clone(*bindings)
		bindings = &sb
	} else {
		bindings = &map[reflect.Type]intentBinding{}
	}
	for i := 0; i < typ.NumMethod(); i++ {
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
			binding := intentBinding{refIdx: refIdx, errIdx: errIdx}
			for i := 1; i < 2 && i < numArgs; i++ {
				if lateApplyType.In(i) == handleCtxType {
					if binding.ctxIdx > 0 {
						return nil, &MethodBindingError{&method,
							fmt.Errorf("intent: %v duplicate HandleContext arg at index %v and %v",
								typ, binding.ctxIdx, i)}
					}
					binding.ctxIdx = i
					skip++
				}
			}
			args := make([]arg, numArgs-skip)
			if err := buildDependencies(lateApplyType, skip, numArgs, args, 0); err != nil {
				err = fmt.Errorf("intent: %v %q: %w", typ, method.Name, err)
				return nil, &MethodBindingError{&method, err}
			}
			binding.funcCall.fun = method.Func
			binding.funcCall.args = args
			(*bindings)[typ] = binding
			intentBindingMap.Store(bindings)
			return &binding, nil
		}
	}
	if require {
		return nil, fmt.Errorf(`intent: %v has no compatible "Apply" method`, typ)
	}
	return nil, nil
}

func (b intentBinding) invoke(
	intent any,
	ctx    HandleContext,
) (promise.Reflect, error) {
	initArgs := []any{intent}
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
	intentBindingLock sync.Mutex
	intentBindingMap   = atomic.Pointer[map[reflect.Type]intentBinding]{}
	intentType         = internal.TypeOf[Intent]()
	promiseReflectType = internal.TypeOf[promise.Reflect]()
)
