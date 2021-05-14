package miruken

import (
	"fmt"
	"reflect"
	"sync"
)

type Binding interface {
	Matches(
		constraint interface{},
		variance   Variance,
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

	AcceptResults(
		results []interface{},
	) (result HandleResult)

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
) (result HandleResult) {
	return NotHandled
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
) (result HandleResult) {
	if len(results) == 0 {
		return Handled
	}
	switch result := results[len(results)-1].(type) {
	case error: return NotHandled.WithError(result)
	case HandleResult: return result
	default: return Handled
	}
}

func (p *contravariantPolicy) NewBinding(
	handlerType  reflect.Type,
	method      *reflect.Method,
) (binding Binding, valid bool) {
	if method == nil {
		return nil, false
	}

	methodType := method.Type
	numArgs    := methodType.NumIn()
	args       := make([]arg, numArgs)

	// Receiver type must match handler
	if methodType.In(0) == handlerType {
		args[0] = _receiverArg
	} else {
		return nil, false
	}

	// Policy argument must be present
	if isPolicy(methodType.In(1)) {
		args[1] = _zeroArg
	} else {
		return nil, false
	}

	// Callback argument must be present
	if numArgs > 2 {
		args[2] = _callbackArg
	} else {
		return nil, false
	}

	for i := 3; i < numArgs; i++ {
		switch at := methodType.In(i); {
		case at ==_handlerContextType:
			args[i] = _handleCtxArg
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
			return b.callbackType.AssignableTo(t)
		case Contravariant:
			return t.AssignableTo(b.callbackType)
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
	if args, err := b.resolveArgs(
		b.args, receiver, callback, rawCallback, ctx); err != nil {
		panic(err)
	} else {
		res := b.method.Func.Call(args)
		results = make([]interface{}, len(res))
		for i, v := range res {
			results[i] = v.Interface()
		}
		return results
	}
}

func (b *methodBinding) resolveArgs(
	args        []arg,
	receiver    interface{},
	callback    interface{},
	rawCallback interface{},
	ctx         HandleContext,
) ([]reflect.Value, error) {
	var resolved []reflect.Value
	for i, arg := range args {
		t := b.method.Type.In(i)
		if a, err := arg.Resolve(t, receiver, callback, rawCallback, ctx); err != nil {
			return nil, err
		} else {
			resolved = append(resolved, a)
		}
	}
	return resolved, nil
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