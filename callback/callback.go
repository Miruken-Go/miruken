package callback

import "reflect"

type Callback interface {
	GetResultType() reflect.Type
	GetResult()     interface{}
	SetResult(result interface{})
}

type CallbackDispatcher interface {
	GetPolicy() Policy

 	Dispatch(
		handler  interface{},
		greedy   bool,
		context  HandleContext,
	) HandleResult
}

type ResultReceiver interface {
	ReceiveResult(
		result   interface{},
		strict   bool,
		greedy   bool,
		context  HandleContext,
	) bool
}

type ResultReceiverFunc func(
	result   interface{},
	strict   bool,
	greedy   bool,
	context  HandleContext,
) bool

func (f ResultReceiverFunc) ResultReceiverFunc(
	result   interface{},
	strict   bool,
	greedy   bool,
	context  HandleContext,
) bool {
	return f(result, strict, greedy, context)
}