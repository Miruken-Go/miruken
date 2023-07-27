package miruken

import "container/list"

type (
	// Policy manages behaviors and callback Binding's.
	Policy interface {
		Filtered
		Strict() bool
		Less(binding, otherBinding Binding) bool
		VariantKey(key any) (bool, bool)
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

	// policyBindings maintains Binding's for a Policy.
	policyBindings struct {
		policy    Policy
		variant   list.List
		index     map[any]*list.Element
		invariant map[any][]Binding
	}

	// policyBindingsMap maps Policy's to policyBindings.
	policyBindingsMap map[Policy]*policyBindings
)


func (p *policyBindings) insert(binding Binding) {
	key := binding.Key()
	if variant, unknown := p.policy.VariantKey(key); variant {
		indexedElem := p.index[key]
		if unknown {
			elem := p.variant.PushBack(binding)
			if indexedElem == nil {
				p.index[key] = elem
			}
			return
		}
		insert := indexedElem
		if insert == nil {
			insert = p.variant.Front()
		}
		for insert != nil && !p.policy.Less(binding, insert.Value.(Binding)) {
			insert = insert.Next()
		}
		var elem *list.Element
		if insert != nil {
			elem = p.variant.InsertBefore(binding, insert)
		} else {
			elem = p.variant.PushBack(binding)
		}
		if indexedElem == nil {
			p.index[key] = elem
		}
	} else {
		if p.invariant == nil {
			p.invariant = make(map[any][]Binding)
			p.invariant[key] = []Binding{binding}
		} else {
			bindings := append(p.invariant[key], binding)
			p.invariant[key] = bindings
		}
	}
}

func (p *policyBindings) reduce(
	key     any,
	reducer BindingReducer,
) (result HandleResult) {
	if reducer == nil {
		panic("reducer cannot be nil")
	}
	done := false
	result = NotHandled
	// Check variant keys (reflect.Type)
	if variant, _ := p.policy.VariantKey(key); variant {
		elem := p.index[key]
		if elem == nil {
			elem = p.variant.Front()
		}
		for elem != nil {
			if result, done = reducer(elem.Value.(Binding), result); done {
				break
			}
			elem = elem.Next()
		}
		return result
		// Check invariant keys (string)
	} else if p.invariant != nil {
		if bs := p.invariant[key]; bs != nil {
			for _, b := range bs {
				if result, done = reducer(b, result); done {
					return result
				}
			}
		}
	}
	// Check unknown keys (any)
	if unk := p.index[anyType]; unk != nil {
		for unk != nil {
			if result, done = reducer(unk.Value.(Binding), result); done {
				break
			}
			unk = unk.Next()
		}
	}
	return result
}

func (p policyBindingsMap) forPolicy(policy Policy) *policyBindings {
	bindings, found := p[policy]
	if !found {
		bindings = &policyBindings{
			policy: policy,
			index:  make(map[any]*list.Element),
		}
		p[policy] = bindings
	}
	return bindings
}


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
		if d := factory.Get(handler); d != nil {
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
