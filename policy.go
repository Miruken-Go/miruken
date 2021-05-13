package miruken

import (
	"fmt"
	"reflect"
	"sync"
)

type Binding interface {
	Matches(
		constraint interface{},
		variance Variance,
	) (matched bool)

	Invoke(
		policy      Policy,
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		ctx         HandleContext,
	) (results []interface{})
}

type Policy interface {
	Variance() Variance

	Constraint(
		callback interface{},
	) reflect.Type

	AcceptResults(results []interface{}) bool

	NewBinding(
		handlerType  reflect.Type,
		method      *reflect.Method, // nil for ctor
	) (binding Binding, valid bool)
}

// covariantPolicy

type covariantPolicy struct{}

func (p *covariantPolicy) Variance() Variance {
	return Covariant
}

func (p *covariantPolicy) Constraint(
	callback interface{},
) reflect.Type {
	return nil
}

func (p *covariantPolicy) AcceptResults(
	results []interface{},
) (accepted bool) {
	return false
}

func (p *covariantPolicy) NewBinding(
	handlerType  reflect.Type,
	method      *reflect.Method,
) (binding Binding, valid bool) {
	return nil, false
}

// contravariantPolicy

type contravariantPolicy struct{}

func (p *contravariantPolicy) Variance() Variance {
	return Contravariant
}

func (p *contravariantPolicy) Constraint(
	callback interface{},
) reflect.Type {
	switch t := callback.(type) {
	case reflect.Type: return t
	default: return reflect.TypeOf(callback)
	}
}

func (p *contravariantPolicy) AcceptResults(
	results []interface{},
) (accepted bool) {
	return false
}

func (p *contravariantPolicy) NewBinding(
	handlerType  reflect.Type,
	method      *reflect.Method,
) (binding Binding, valid bool) {
	if method == nil {
		return nil, false
	}

	var args []arg
	methodType := method.Type
	numArgs    := methodType.NumIn()

	// Receiver type must match handler
	if methodType.In(0) == handlerType {
		args = append(args, _receiverArg)
	} else {
		return nil, false
	}

	// Policy argument must be present
	if isPolicy(methodType.In(1)) {
		args = append(args, _policyArg)
	} else {
		return nil, false
	}

	// Callback argument must be present
	if numArgs > 2 {
		args = append(args, callbackArg{methodType.In(2)})
	} else {
		return nil, false
	}

	for i := 3; i < numArgs; i++ {
		switch at := methodType.In(i); {
		case at ==_handlerContextType:
			args = append(args, _handleCtxArg)
		default:
			// TODO: Dependencies coming soon
			return nil, false
		}
	}

	return &methodBinding{
		callbackType: methodType.In(2),
		method:      *method,
		args:         args,
	}, true
}

// methodBinding

type methodBinding struct {
	callbackType reflect.Type
	method       reflect.Method
	args         []arg
}

func (b *methodBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	if t, ok := constraint.(reflect.Type); ok {
		switch variance {
		case Covariant:
			return t.AssignableTo(b.callbackType)
		case Contravariant:
			return b.callbackType.AssignableTo(t)
		}
	}
	return false
}

func (b *methodBinding) Invoke(
	policy      Policy,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
)  (results []interface{}) {
	if args, err := resolveArgs(
		b.args, policy, receiver, callback, rawCallback, ctx); err != nil {
		panic(err)
	} else {
		r := b.method.Func.Call(args)
		results = make([]interface{}, len(r))
		for i, v := range r {
			results[i] = v.Interface()
		}
		return results
	}
}

// constructorBinding

type constructorBinding struct {
	handlerType reflect.Type
}

func (b *constructorBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	return false
}

func (b *constructorBinding) Invoke(
	policy      Policy,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) (results []interface{}) {
	return nil
}

func DispatchPolicy(
	policy      Policy,
	handler     interface{},
	callback    interface{},
	rawCallback interface{},
	greedy      bool,
	ctx         HandleContext,
	results     ResultReceiver,
) HandleResult {
	if factory := GetHandlerDescriptorFactory(ctx); factory != nil {
		handlerType := reflect.TypeOf(handler)
		if d, err := factory.GetHandlerDescriptor(handlerType); d != nil {
			if rawCallback == nil {
				rawCallback = callback
			}
			return d.Dispatch(policy, handler, callback, rawCallback, greedy, ctx, results)
		} else if err != nil {
			return NotHandled.WithError(err)
		}
	}

	return NotHandled
}

func RegisterPolicy(policy Policy) Policy {
	if policy == nil {
		panic("nil policy")
	}
	policyType := reflect.TypeOf(policy).Elem()
	if _, loaded := _policies.LoadOrStore(policyType, policy); loaded {
		panic(fmt.Sprintf("policy: %v already registered", policyType))
	}
	return policy
}

func isPolicy(t reflect.Type) bool {
	return reflect.PtrTo(t).Implements(_policyType)
}

func requirePolicy(policyType reflect.Type) Policy {
	if policy, ok := _policies.Load(policyType); ok {
		return policy.(Policy)
	}
	panic(fmt.Sprintf("policy: %v not found.  Did you forget to call RegisterPolicy?", policyType))
}

// Standard _policies

var (
	_policies sync.Map
	_policyType = reflect.TypeOf((*Policy)(nil)).Elem()
	_handles    = RegisterPolicy(new(Handles))
	_provides   = RegisterPolicy(new(Provides))
	_creates    = RegisterPolicy(new(Creates))
)

// Handles policy for handling callbacks contravariantly.
type Handles struct {
	contravariantPolicy
}
func HandlesPolicy() Policy { return _handles }

// Provides policy for providing instances covariantly.
type Provides struct {
	covariantPolicy
}
func ProvidesPolicy() Policy { return _provides }

// Creates policy for creating instances covariantly.
type Creates struct {
	covariantPolicy
}
func CreatesPolicy() Policy { return _creates }