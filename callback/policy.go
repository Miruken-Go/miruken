package callback

import "miruken.com/miruken"

type Policy interface {
	GetVariance() miruken.Variance
}

type PolicyDispatcher interface {
	Dispatch(
		policy   Policy,
		callback interface{},
		greedy   bool,
		context  HandleContext,
		results  ResultsFunc,
	) (HandleResult, error)
}

func DispatchPolicy(
	policy   Policy,
	handler  interface{},
	callback interface{},
	greedy   bool,
	context  HandleContext,
	results  ResultsFunc,
) (HandleResult, error) {
	if dispatch, ok := handler.(PolicyDispatcher); ok {
		return dispatch.Dispatch(policy, callback, greedy, context, results)
	}

	//descriptor, err := GetDescriptor(handler)

	//if err != nil {

	//}

	return NotHandled, nil
}

var (
	HandlesPolicy  = new(Handles)
	ProvidesPolicy = new(Provides)
	CreatesPolicy  = new(Creates)
)

// Handles policy for handling callbacks contravariantly.
type Handles struct{}

func (h Handles) GetVariance() miruken.Variance {
	return miruken.Contravariant
}

// Provides policy for providing instances covariantly.
type Provides struct{}

func (p Provides) GetVariance() miruken.Variance {
	return miruken.Covariant
}

// Creates policy for creating instances covariantly.
type Creates struct{}

func (c Creates) GetVariance() miruken.Variance {
	return miruken.Covariant
}
