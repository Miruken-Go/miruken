package callback

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
		result interface{},
		strict bool,
		greedy bool,
		ctx    HandleContext,
	) bool
}

type ResultReceiverFunc func(
	result interface{},
	strict bool,
	greedy bool,
	ctx    HandleContext,
) bool

func (f ResultReceiverFunc) ResultReceiverFunc(
	result interface{},
	strict bool,
	greedy bool,
	ctx    HandleContext,
) bool {
	return f(result, strict, greedy, ctx)
}