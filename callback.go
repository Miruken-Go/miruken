package miruken

import "reflect"

type Callback interface {
	ResultType() reflect.Type
	SetResult(result interface{})
	Result()     interface{}
}

type CallbackDispatcher interface {
	Policy() Policy

 	Dispatch(
		handler interface{},
		greedy  bool,
		ctx     HandleContext,
	) HandleResult
}

type ResultReceiver interface {
	ReceiveResult(
		results interface{},
		strict  bool,
		greedy  bool,
		ctx     HandleContext,
	) (accepted bool)
}

type ResultReceiverFunc func(
	result interface{},
	strict bool,
	greedy bool,
	ctx    HandleContext,
) (accepted bool)

func (f ResultReceiverFunc) ReceiveResult(
	results interface{},
	strict  bool,
	greedy  bool,
	ctx     HandleContext,
) (accepted bool) {
	return f(results, strict, greedy, ctx)
}