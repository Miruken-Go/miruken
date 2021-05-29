package miruken

import "reflect"

type InferenceHandler struct {
	handlerTypes []reflect.Type
}

func (h *InferenceHandler) Handle(
	callback interface{},
	greedy   bool,
	composer Handler,
) HandleResult {
	return NotHandled
}

func NewInferenceHandler(types ... reflect.Type) *InferenceHandler {
	return &InferenceHandler{types}
}