package miruken

import (
	"container/list"
	"iter"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/internal/seq"
)

type (
	// Policy manages behaviors and callback Binding's.
	Policy interface {
		Filtered
		Strict() bool
		Less(binding, otherBinding Binding) bool
		VariantKey(key any) (bool, bool)
		MatchesKey(key, otherKey any, invariant bool) (bool, bool)
		AcceptResults(results []any) (any, HandleResult, []Effect, []any)
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

	// indexedBindingList maintains a Binding list ordered by Policy.
	// It keeps a linked list of bindings that are partially
	// ordered for keys that support variance (reflect.Type).
	// An index into the list is added to optimize lookups.
	// An invariant map stores the bindings for exact keys (string).
	// A dynamic index is also maintained interface relationships
	// which can only be determined at runtime.
	indexedBindingList struct {
		variant   list.List
		invariant map[any][]Binding
		index     map[any]*list.Element
		dynIndex  atomic.Pointer[map[any]*list.Element]
		dynLock   sync.Mutex
	}

	// policyBindingMap maintains a maps of Policy keys to indexedBindingList.
	// It is the primary data-structure to enabled Callback dispatch.
	policyBindingMap map[Policy]*indexedBindingList
)

func (b *indexedBindingList) all() iter.Seq[Binding] {
	return func(yield func(Binding) bool) {
		for e := b.variant.Front(); e != nil; e = e.Next() {
			if !yield(e.Value.(Binding)) {
				return
			}
		}
		for _, bs := range b.invariant {
			for _, b := range bs {
				if !yield(b) {
					return
				}
			}
		}
	}
}

func (b *indexedBindingList) insert(policy Policy, binding Binding) {
	key := binding.Key()
	if variant, unknown := policy.VariantKey(key); variant {
		indexedElem := b.index[key]
		if unknown {
			elem := b.variant.PushBack(binding)
			if indexedElem == nil {
				b.index[key] = elem
			}
			return
		}
		insert := indexedElem
		if insert == nil {
			insert = b.variant.Front()
		}
		for insert != nil && !policy.Less(binding, insert.Value.(Binding)) {
			insert = insert.Next()
		}
		var elem *list.Element
		if insert != nil {
			elem = b.variant.InsertBefore(binding, insert)
		} else {
			elem = b.variant.PushBack(binding)
		}
		if indexedElem == nil {
			b.index[key] = elem
		}
	} else {
		if b.invariant == nil {
			b.invariant = make(map[any][]Binding)
			b.invariant[key] = []Binding{binding}
		} else {
			b.invariant[key] = append(b.invariant[key], binding)
		}
	}
}

func (b *indexedBindingList) reduce(
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
	if variant, unknown := policy.VariantKey(key); variant {
		var needsIndex bool
		elem := b.index[key]
		if elem == nil {
			if !unknown {
				if dynIndex := b.dynIndex.Load(); dynIndex != nil {
					elem = (*dynIndex)[key]
				}
			}
			if elem == nil {
				elem = b.variant.Front()
				needsIndex = true
			}
		}
		for elem != nil {
			binding := elem.Value.(Binding)
			if result, done = reducer(binding, result); done {
				if needsIndex {
					// Since interfaces implemented by a type are implied
					// and cannot be enumerated, we need to dynamically index
					// the binding for future lookups.
					// Uses the copy-on-write idiom since reads should be more
					// frequent than writes.
					needsIndex = false
					b.dynLock.Lock()
					dynIndex := b.dynIndex.Load()
					if dynIndex != nil {
						if _, ok := (*dynIndex)[key]; !ok {
							di := maps.Clone(*dynIndex)
							di[key] = elem
							dynIndex = &di
						}
					} else {
						dynIndex = &map[any]*list.Element{key: elem}
					}
					b.dynIndex.Store(dynIndex)
					b.dynLock.Unlock()
				}
				break
			}
			elem = elem.Next()
		}
		return result
	} else if b.invariant != nil {
		// Check invariant keys (string)
		if bs := b.invariant[key]; bs != nil {
			for _, b := range bs {
				if result, done = reducer(b, result); done {
					return result
				}
			}
		}
	}
	// Check unknown keys (any)
	if unk := b.index[internal.AnyType]; unk != nil {
		for unk != nil {
			if result, done = reducer(unk.Value.(Binding), result); done {
				break
			}
			unk = unk.Next()
		}
	}
	return result
}

func (p policyBindingMap) forPolicy(policy Policy) *indexedBindingList {
	bindings, found := p[policy]
	if !found {
		bindings = &indexedBindingList{
			index: make(map[any]*list.Element),
		}
		p[policy] = bindings
	}
	return bindings
}

func (p policyBindingMap) insert(policy Policy, binding Binding) {
	p.forPolicy(policy).insert(policy, binding)
}

func (p policyBindingMap) all() iter.Seq2[Policy, iter.Seq[Binding]] {
	return func(yield func(Policy, iter.Seq[Binding]) bool) {
		for policy, bindings := range p {
			if !yield(policy, bindings.all()) {
				return
			}
		}
	}
}

func (p policyBindingMap) policy(policy Policy) iter.Seq[Binding] {
	if pb, ok := p[policy]; ok {
		return pb.all()
	}
	return seq.Empty[Binding]()
}

func PolicyOf[C Callback]() Policy {
	var callback C
	//goland:noinspection ALL
	return callback.Policy()
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
	if factory := CurrentHandlerRuntimeFactory(composer); factory != nil {
		if info := factory.Get(handler); info != nil {
			return info.Dispatch(policy, handler, callback, greedy, composer, nil)
		}
	}
	return NotHandled
}

var (
	callbackType  = reflect.TypeFor[Callback]()
	handleResType = reflect.TypeFor[HandleResult]()
)
