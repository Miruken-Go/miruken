package miruken

import (
	"fmt"
	"reflect"
)

var (
	_zeroArg       = zeroArg{}
	_callbackArg   = callbackArg{}
	_receiverArg   = receiverArg{}
	_dependencyArg = dependencyArg{}
)

// Manages arguments

type arg interface {
	Resolve(
		typ         reflect.Type,
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		ctx         HandleContext,
	) (reflect.Value, error)
}

// receiverArg

type receiverArg struct {}

func (a receiverArg) Resolve(
	typ         reflect.Type,
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
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	return reflect.Zero(typ), nil
}

// callbackArg

type callbackArg struct {}

func (a callbackArg) Resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	if v := reflect.ValueOf(callback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	if v := reflect.ValueOf(rawCallback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("unable to resolve callback: %v", typ)
}

// dependencyArg

type dependencyArg struct {}

func (a dependencyArg) Resolve(
	typ         reflect.Type,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (reflect.Value, error) {
	if typ == _handlerContextType {
		return reflect.ValueOf(ctx), nil
	}
	if v := reflect.ValueOf(rawCallback); v.Type().AssignableTo(typ) {
		return v, nil
	}
	return reflect.ValueOf(nil), fmt.Errorf("unable to resolve dependency: %v", typ)
}