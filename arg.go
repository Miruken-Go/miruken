package miruken

import (
	"fmt"
	"reflect"
)

var (
	_policyArg    = policyArg{}
	_receiverArg  = receiverArg{}
	_handleCtxArg = handleCtxArg{}
)

// Manages arguments

type arg interface {
	Resolve(
		policy      Policy,
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		ctx         HandleContext,
	) (reflect.Value, error)
}

// receiverArg

type receiverArg struct {}

func (a receiverArg) Resolve(
	_        Policy,
	receiver interface{},
	_        interface{},
	_        interface{},
	_        HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(receiver), nil
}

// policyArg

type policyArg struct {}

func (a policyArg) Resolve(
	policy   Policy,
	_        interface{},
	_        interface{},
	_        interface{},
	_        HandleContext,
) (reflect.Value, error) {
	value := reflect.ValueOf(policy)
	if value.Type().Kind() == reflect.Ptr {
		return reflect.Indirect(value), nil
	}
	return value, nil
}

// callbackArg

type callbackArg struct {
	t reflect.Type
}

func (a callbackArg) Resolve(
	_           Policy,
	_           interface{},
	callback    interface{},
	rawCallback interface{},
	_           HandleContext,
) (reflect.Value, error) {
	if v := reflect.ValueOf(callback); a.t.AssignableTo(v.Type()) {
		return v, nil
	}
	if v := reflect.ValueOf(rawCallback); a.t.AssignableTo(v.Type()) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("unable to resolve callback type: %v", a.t)
}

// handleCtxArg

type handleCtxArg struct {}

func (a handleCtxArg) Resolve(
	_        Policy,
	_        interface{},
	_        interface{},
	_        interface{},
	ctx      HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(ctx), nil
}

func resolveArgs(
	args        []arg,
	policy      Policy,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) ([]reflect.Value, error) {
	var resolved []reflect.Value
	for _, arg := range args {
		if a, err := arg.Resolve(policy, receiver, callback, rawCallback, ctx); err != nil {
			return nil, err
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
}