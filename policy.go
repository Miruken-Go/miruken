package miruken

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"sync"
)

// Binding

type Binding interface {
	Constraint() interface{}

	Matches(
		constraint interface{},
		variance   Variance,
	) (matched bool)

	Invoke(
		receiver    interface{},
		callback    interface{},
		rawCallback interface{},
		ctx         HandleContext,
	) (results []interface{})
}

// Policy

type Policy interface {
	OrderBinding
	Variance() Variance

	Constraint(
		callback interface{},
	) reflect.Type

	AcceptResults(
		results []interface{},
	) (result interface{}, accepted HandleResult)
}

// MethodBinder

type MethodBinder interface {
	NewMethodBinding(
		method  reflect.Method,
	) (binding Binding, invalid error)
}

type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: reason %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

func (e *MethodBindingError) Unwrap() error { return e.Reason }

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
) (result interface{}, accepted HandleResult) {
	return nil, NotHandled
}

func (p *covariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	constraint := binding.Constraint()
	if otherBinding.Matches(constraint, Invariant) {
		return false
	} else if otherBinding.Matches(constraint, Covariant) {
		return true
	}
	return false
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
) (result interface{}, accepted HandleResult) {
	switch len(results) {
	case 0:
		return nil, Handled
	case 1:
		switch result := results[0].(type) {
		case error:
			return nil, NotHandled.WithError(result)
		case HandleResult:
			return nil, result
		default:
			return result, Handled
		}
	case 2:
		switch result := results[1].(type) {
		case error:
			return results[0], NotHandled.WithError(result)
		case HandleResult:
			return results[0], result
		}
	}
	return nil, NotHandled.WithError(
		errors.New("contravariant policy: cannot accept more than 2 results"))
}

func (p *contravariantPolicy) Less(
	binding, otherBinding Binding,
) bool {
	if binding == nil {
		panic("binding cannot be nil")
	}
	if otherBinding == nil {
		panic("otherBinding cannot be nil")
	}
	constraint := binding.Constraint()
	if otherBinding.Matches(constraint, Invariant) {
		return false
	} else if otherBinding.Matches(constraint, Contravariant) {
		return true
	}
	return false
}

func (p *contravariantPolicy) NewMethodBinding(
	method  reflect.Method,
) (binding Binding, invalid error) {
	methodType := method.Type
	numArgs    := methodType.NumIn()
	args       := make([]arg, numArgs)

	args[0] = _receiverArg
	args[1] = _zeroArg  // policy marker

	// Callback argument must be present
	if numArgs > 2 {
		args[2] = _callbackArg
	} else {
		invalid = multierror.Append(invalid,
			errors.New("contravariant policy: missing callback argument"))
	}

	for i := 3; i < numArgs; i++ {
		switch at := methodType.In(i); {
		case at ==_handlerContextType:
			args[i] = _handleCtxArg
		default:
			// TODO: Dependencies coming soon
			invalid = multierror.Append(invalid,
				errors.New("contravariant policy: additional dependencies are not supported yet"))
		}
	}

	switch methodType.NumOut() {
	case 0, 1: break
	case 2:
		switch methodType.Out(1) {
		case _errorType, _handleResType: break
		default:
			invalid = multierror.Append(invalid,
				fmt.Errorf("contravariant policy: when two return values, second must be %v or %v",
					_errorType, _handleResType))
		}
	default:
		invalid = multierror.Append(invalid,
			fmt.Errorf("contravariant policy: at most two return values allowed and second must be %v or %v",
				_errorType, _handleResType))
	}

	if invalid != nil {
		return nil, &MethodBindingError{method, invalid}
	}

	return &methodBinding{
		callbackType: methodType.In(2),
		method:       method,
		args:         args,
	}, nil
}

// methodBinding

type methodBinding struct {
	callbackType reflect.Type
	method       reflect.Method
	args         []arg
}

func (b *methodBinding) Constraint() interface{} {
	return b.callbackType
}

func (b *methodBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	if t, ok := constraint.(reflect.Type); ok {
		if t == b.callbackType {
			return true
		}
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
		typ := b.method.Type.In(i)
		if a, err := arg.Resolve(typ, receiver, callback, rawCallback, ctx); err != nil {
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
		panic("policy cannot be nil")
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
	_policyType    = reflect.TypeOf((*Policy)(nil)).Elem()
	_handleResType = reflect.TypeOf((*HandleResult)(nil)).Elem()
	_errorType     = reflect.TypeOf((*error)(nil)).Elem()
	_handles       = RegisterPolicy(new(Handles))
	_provides      = RegisterPolicy(new(Provides))
	_creates       = RegisterPolicy(new(Creates))
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