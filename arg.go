package miruken

import (
	"fmt"
	"reflect"
)

var (
	_zeroArg      = zeroArg{}
	_callbackArg  = callbackArg{}
	_receiverArg  = receiverArg{}
	_handleCtxArg = handleCtxArg{}
)

// Manages arguments

type arg interface {
	Resolve(
		t           reflect.Type,
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		ctx         HandleContext,
	) (reflect.Value, error)
}

// receiverArg

type receiverArg struct {}

func (a receiverArg) Resolve(
	t           reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(receiver), nil
}

// zeroArg

type zeroArg struct {}

func (a zeroArg) Resolve(
	t           reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	return reflect.Zero(t), nil
}

// callbackArg

type callbackArg struct {}

func (a callbackArg) Resolve(
	t           reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	if v := reflect.ValueOf(callback); v.Type().AssignableTo(t) {
		return v, nil
	}
	if v := reflect.ValueOf(rawCallback); v.Type().AssignableTo(t) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("unable to resolve callback type: %v", t)
}

// handleCtxArg

type handleCtxArg struct {}

func (a handleCtxArg) Resolve(
	t           reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	return reflect.ValueOf(ctx), nil
}
