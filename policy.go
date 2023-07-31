package miruken

import (
	"container/list"
	"github.com/miruken-go/miruken/internal"
)

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

	// policyInfo maintains Binding's for a Policy.
	// It keeps a linked list of bindings that are partially
	// ordered for keys that support variance (reflect.Type).
	// An index into the list is used to optimize lookups.
	// An invariant map stores the bindings for exact keys (string).
	policyInfo struct {
		variant   list.List
		index     map[any]*list.Element
		invariant map[any][]Binding
	}

	// policyInfoMap maps Policy instances to policyInfo.
	policyInfoMap map[Policy]*policyInfo
)


func (p *policyInfo) insert(policy Policy, binding Binding) {
	key := binding.Key()
	if variant, unknown := policy.VariantKey(key); variant {
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
		for insert != nil && !policy.Less(binding, insert.Value.(Binding)) {
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

func (p *policyInfo) reduce(
	key     any,
	policy  Policy,
	reducer BindingReducer,
) (result HandleResult) {
	if reducer == nil {
		panic("reducer cannot be nil")
	}
	done := false
	result = NotHandled
	// Check variant keys (reflect.Type)
	if variant, _ := policy.VariantKey(key); variant {
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

func (p policyInfoMap) forPolicy(policy Policy) *policyInfo {
	bindings, found := p[policy]
	if !found {
		bindings = &policyInfo{
			index: make(map[any]*list.Element),
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
	if factory := CurrentHandlerInfoFactory(composer); factory != nil {
		if d := factory.Get(handler); d != nil {
			return d.Dispatch(policy, handler, callback, greedy, composer, nil)
		}
	}
	return NotHandled
}


var (
	anyType       = internal.TypeOf[any]()
	anySliceType  = internal.TypeOf[[]any]()
	errorType     = internal.TypeOf[error]()
	callbackType  = internal.TypeOf[Callback]()
	handleResType = internal.TypeOf[HandleResult]()
)
