package miruken

type (
	// Policy manages behaviors and callback Binding's.
	Policy interface {
		Filtered
		Strict() bool
		Less(binding, otherBinding Binding) bool
		IsVariantKey(key any) (bool, bool)
		MatchesKey(key, otherKey any, invariant bool) (bool, bool)
		AcceptResults(results []any) (any, HandleResult)
	}

	// PolicyDispatch customizes Callback Policy dispatch.
	PolicyDispatch interface {
		DispatchPolicy(
			policy   Policy,
			callback Callback,
			greedy   bool,
			composer Handler,
		) HandleResult
	}

	// policyKey binds a Policy to a key for lookup.
	policyKey struct {
		policy Policy
		key    any
	}
)

func DispatchPolicy(
	handler  any,
	callback Callback,
	greedy   bool,
	composer Handler,
) HandleResult {
	policy := callback.Policy()
	if dp, ok := handler.(PolicyDispatch); ok {
		return dp.DispatchPolicy(policy, callback, greedy, composer)
	}
	if factory := CurrentHandlerDescriptorFactory(composer); factory != nil {
		if d := factory.Descriptor(handler); d != nil {
			return d.Dispatch(policy, handler, callback, greedy, composer, nil)
		}
	}
	return NotHandled
}


var (
	anyType       = TypeOf[any]()
	anySliceType  = TypeOf[[]any]()
	errorType     = TypeOf[error]()
	callbackType  = TypeOf[Callback]()
	handleResType = TypeOf[HandleResult]()
)
