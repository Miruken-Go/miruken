package callback

type HandlerDescriptor struct {

}

func (d *HandlerDescriptor) Dispatch(
	policy   Policy,
	callback interface{},
	greedy   bool,
	composer Handler,
	results  ResultsFunc,
	) (HandleResult, error) {
	return NotHandled, nil
}

func GetDescriptor(handler interface{}) (*HandlerDescriptor, error) {
	return nil, nil
}