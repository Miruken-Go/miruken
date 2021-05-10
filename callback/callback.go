package callback

import "reflect"

type ResultsFunc func(
	result   interface{},
	strict   bool,
	greedy   bool,
	composer Handler)

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