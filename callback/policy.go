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
		results  ResultReceiver,
	) HandleResult
}

func DispatchPolicy(
	policy   Policy,
	handler  interface{},
	callback interface{},
	greedy   bool,
	context  HandleContext,
	results  ResultReceiver,
) HandleResult {
	if dispatch, ok := handler.(PolicyDispatcher); ok {
		return dispatch.Dispatch(policy, callback, greedy, context, results)
	}

	if factory := GetHandlerDescriptorFactory(context); factory != nil {
		if descriptor := factory.GetHandlerDescriptor(handler); descriptor != nil {
			return descriptor.Dispatch(policy, callback, greedy, context, results)
		}
	}

	return NotHandled
}

var (
	handles  = new(Handles)
	provides = new(Provides)
	creates  = new(Creates)
)

// Handles policy for handling callbacks contravariantly.
type Handles struct{}

func (h Handles) GetVariance() miruken.Variance {
	return miruken.Contravariant
}

func GetHandlesPolicy() Policy { return handles }

// Provides policy for providing instances covariantly.
type Provides struct{}

func (p Provides) GetVariance() miruken.Variance {
	return miruken.Covariant
}

func GetProvidesPolicy() Policy { return provides }

// Creates policy for creating instances covariantly.
type Creates struct{}

func (c Creates) GetVariance() miruken.Variance {
	return miruken.Covariant
}

func GetCreatesPolicy() Policy { return creates }