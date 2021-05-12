package callback

import (
	"fmt"
	"miruken.com/miruken"
	"reflect"
	"sync"
)

type Policy interface {
	Variance() miruken.Variance
	BindingFor(method reflect.Method) Binding
}

type PolicyDispatcher interface {
	Dispatch(
		policy   Policy,
		callback interface{},
		greedy   bool,
		ctx      HandleContext,
		results  ResultReceiver,
	) HandleResult
}

// CovariantPolicy

type CovariantPolicy struct{}

func (p *CovariantPolicy) Variance() miruken.Variance {
	return miruken.Covariant
}

func (p *CovariantPolicy) BindingFor(method reflect.Method) Binding {
	return &emptyBinding{}
}

// ContravariantPolicy

type ContravariantPolicy struct{}

func (p *ContravariantPolicy) Variance() miruken.Variance {
	return miruken.Contravariant
}

func (p *ContravariantPolicy) BindingFor(method reflect.Method) Binding {
	return &emptyBinding{}
}

func DispatchPolicy(
	policy   Policy,
	handler  interface{},
	callback interface{},
	greedy   bool,
	ctx      HandleContext,
	results  ResultReceiver,
) HandleResult {
	if dispatch, ok := handler.(PolicyDispatcher); ok {
		return dispatch.Dispatch(policy, callback, greedy, ctx, results)
	}

	if factory := GetHandlerDescriptorFactory(ctx); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if descriptor := factory.GetHandlerDescriptor(handlerType); descriptor != nil {
			return descriptor.Dispatch(policy,handler, callback, greedy, ctx, results)
		}
	}

	return NotHandled
}

func RegisterPolicy(policy Policy) Policy {
	if policy == nil {
		panic("nil policy")
	}
	policyType := reflect.TypeOf(policy).Elem()
	if _, ok := policies.Load(policyType); ok {
		panic(fmt.Sprintf("policy: %v already registered", policyType))
	}
	policies.Store(policyType, policy)
	return policy
}

func requirePolicy(policyType reflect.Type) Policy {
	if policy, ok := policies.Load(policyType); ok {
		return policy.(Policy)
	}
	panic(fmt.Sprintf("policy: %v not found.  Did you forget to call RegisterPolicy?", policyType))
}

var (
	policies sync.Map
	handles  = RegisterPolicy(new(Handles))
	provides = RegisterPolicy(new(Provides))
	creates  = RegisterPolicy(new(Creates))
)

// Handles policy for handling callbacks contravariantly.
type Handles struct {
	ContravariantPolicy
}
func HandlesPolicy() Policy { return handles }

// Provides policy for providing instances covariantly.
type Provides struct {
	CovariantPolicy
}
func ProvidesPolicy() Policy { return provides }

// Creates policy for creating instances covariantly.
type Creates struct {
	CovariantPolicy
}
func CreatesPolicy() Policy { return creates }