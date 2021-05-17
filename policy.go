package miruken

import (
	"errors"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/hashicorp/go-multierror"
	"reflect"
	"strconv"
	"sync"
)

// Binding

type Binding interface {
	Strict()     bool
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

	AcceptResults(
		results []interface{},
	) (result interface{}, accepted HandleResult)
}

// methodBinder

type MethodBindingError struct {
	Method reflect.Method
	Reason error
}

func (e *MethodBindingError) Error() string {
	return fmt.Sprintf("invalid method: %v %v: reason %v",
		e.Method.Name, e.Method.Type, e.Reason)
}

type methodBinder interface {
	newMethodBinding(
		method  reflect.Method,
		spec   *bindingSpec,
	) (binding Binding, invalid error)
}

func (e *MethodBindingError) Unwrap() error { return e.Reason }

// covariantPolicy

type covariantPolicy struct{}

func (p *covariantPolicy) Variance() Variance {
	return Covariant
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

func (p *covariantPolicy) newMethodBinding(
	method  reflect.Method,
	spec   *bindingSpec,

) (binding Binding, invalid error) {
	return nil, nil
}

// contravariantPolicy

type contravariantPolicy struct{}

func (p *contravariantPolicy) Variance() Variance {
	return Contravariant
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

func (p *contravariantPolicy) newMethodBinding(
	method  reflect.Method,
	spec   *bindingSpec,

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
		spec:         spec,
		callbackType: methodType.In(2),
		method:       method,
		args:         args,
	}, nil
}

// bindingSpec

type bindingSpec struct {
	strict bool
}

// methodBinding

type methodBinding struct {
	spec         *bindingSpec
	callbackType  reflect.Type
	method        reflect.Method
	args          []arg
}

func (b *methodBinding) Strict() bool {
	return b.spec != nil && b.spec.strict
}

func (b *methodBinding) Constraint() interface{} {
	return b.callbackType
}

func (b *methodBinding) Matches(
	constraint interface{},
	variance   Variance,
) (matched bool) {
	if typ, ok := constraint.(reflect.Type); ok {
		if typ == b.callbackType {
			return true
		}
		switch variance {
		case Covariant:
			return b.callbackType.AssignableTo(typ)
		case Contravariant:
			return typ.AssignableTo(b.callbackType)
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
	constraint  interface{},
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
			return d.Dispatch(
				policy, handler, callback, rawCallback,
				constraint, greedy, ctx, results)
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

func isPolicy(typ reflect.Type) bool {
	return reflect.PtrTo(typ).Implements(_policyType)
}

func getPolicy(policyType reflect.Type) Policy {
	if policy, ok := _policies.Load(policyType); ok {
		return policy.(Policy)
	}
	return nil
}

func inferBinding(
	bindingType reflect.Type,
) (policy Policy, spec *bindingSpec, err error) {
	var policyType reflect.Type
	// Is it a policy type already?
	if isPolicy(bindingType) {
		policyType = bindingType
		if policy = getPolicy(policyType); policy != nil {
			return policy, nil, nil
		}
	}
	// Is it a binding specification?
	if bindingType.Kind() == reflect.Ptr {
		bindingType = bindingType.Elem()
	}
	if bindingType.Kind() == reflect.Struct && bindingType.NumField() > 0 {
		field := bindingType.Field(0)
		if isPolicy(field.Type) {
			policyType = field.Type
			if policy = getPolicy(policyType); policy != nil {
				spec, err := parseBindingSpec(bindingType)
				return policy, spec, err
			}
		}
	}
	if policyType != nil {
		panic(fmt.Sprintf("policy: %v not found.  Did you forget to call RegisterPolicy?", policyType))
	}
	return nil, nil, nil
}

type parserFunc func (reflect.StructField, int, *bindingSpec) (bool, error)
var parsers = []parserFunc{parsePolicy}

func parseBindingSpec(
	specType reflect.Type,
) (spec *bindingSpec, err error) {
	spec = new(bindingSpec)
	for i := 0; i < specType.NumField(); i++ {
		for _, parser := range parsers {
			field := specType.Field(i)
			if matched, invalid := parser(field, i, spec); matched {
				continue
			} else if invalid != nil {
				err = multierror.Append(err, invalid)
			}
		}
	}
	return spec, err
}

func parsePolicy(
	field  reflect.StructField,
	index  int,
	spec  *bindingSpec,
) (bool, error) {
	if index != 0 {
		return false, nil
	}
	tag := field.Tag.Get(_strictTag)
	tags, err := structtag.Parse(string(field.Tag))
	if err != nil {
		return false, fmt.Errorf("binding: invalid tag %v on field %v %w",
			tag, field.Name, err)
	}
	if strictTag, _ := tags.Get("strict"); strictTag != nil {
		if strict, err := strconv.ParseBool(tag); err == nil {
			spec.strict = strict
		} else {
			return false, fmt.Errorf("binding: invalid value %q for %q tag on field %v %w",
				tag, _strictTag, field.Name, err)
		}
	}
	return true, nil
}

// Standard _policies

var (
	_policies sync.Map
	_strictTag     = "strict"
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