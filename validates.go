package miruken

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Validates callbacks contravariantly.
type Validates struct {
	CallbackBase
	target   any
	scopes   []any
	outcome  ValidationOutcome
	metadata BindingMetadata
}

func (v *Validates) Target() any {
	return v.target
}

func (v *Validates) Scopes() []any {
	return v.scopes
}

func (v *Validates) Outcome() *ValidationOutcome {
	return &v.outcome
}

func (v *Validates) Key() any {
	return reflect.TypeOf(v.target)
}

func (v *Validates) Policy() Policy {
	return _validatesPolicy
}

func (v *Validates) Metadata() *BindingMetadata {
	return &v.metadata
}

func (v *Validates) Dispatch(
	handler  any,
	greedy   bool,
	composer Handler,
) HandleResult {
	return DispatchPolicy(handler, v.target, v, greedy, composer)
}

type Scope struct {
	scopes map[any]struct{}
}

func (s *Scope) InitWithTag(tag reflect.StructTag) error {
	if sc, ok := tag.Lookup("scope"); ok {
		s.scopes = make(map[any]struct{})
		if scope := strings.TrimSpace(sc); len(scope) > 0 {
			s.scopes[scope] = struct{}{}
		}
	}
	if len(s.scopes) == 0 {
		return errors.New("the Scope constraint requires a non-empty `scope:scope` tag")
	}
	return nil
}

func (s *Scope) Merge(constraint BindingConstraint) bool {
	if scope, ok := constraint.(*Scope); ok {
		for scope := range scope.scopes {
			s.scopes[scope] = struct{}{}
		}
		return true
	}
	return false
}

func (s *Scope) Require(metadata *BindingMetadata) {
	if scopes := s.scopes; len(scopes) > 0 {
		keys, i := make([]any, len(scopes)), 0
		for key := range scopes {
			keys[i] = key
			i++
		}
		metadata.Set(reflect.TypeOf(s), keys)
	}
}

func (s *Scope) Matches(metadata *BindingMetadata) bool {
	if m, ok := metadata.Get(reflect.TypeOf(s)); ok {
		if scopes, ok := m.([]any); ok {
			for _, scope := range scopes {
				if _, found := s.scopes[scope]; found {
					return true
				}
			}
		}
	}
	return false
}

// ValidationOutcome captures structured validation errors.
type ValidationOutcome struct {
	errors map[string][]error
}

func (v *ValidationOutcome) Valid() bool {
	return v.errors == nil
}

func (v *ValidationOutcome) Culprits() []string {
	var keys []string
	if errs := v.errors; len(errs) > 0 {
		keys = make([]string, len(errs))
		i := 0
		for key := range errs {
			keys[i] = key
			i++
		}
	}
	return keys
}

func (v *ValidationOutcome) AddError(
	path string,
	err  error,
) {
	if len(path) == 0 {
		panic("path cannot be empty")
	}
	if err == nil {
		panic("err cannot be nil")
	}
	if parent, key := v.parsePath(path, true); parent == v {
		if v.errors == nil {
			v.errors = map[string][]error{key: {err}}
		} else {
			v.errors[key] = append(v.errors[key], err)
		}
	} else {
		parent.AddError(key, err)
	}
}

func (v *ValidationOutcome) PathErrors(
	path string,
) []error {
	if len(path) == 0 {
		panic("path cannot be empty")
	}
	var empty []error
	if parent, key := v.parsePath(path, true); parent == nil {
		return empty
	} else if parent != v {
		return parent.PathErrors(key)
	}
	if v.errors != nil {
		if errs, found := v.errors[path]; found {
			return errs
		}
	}
	return empty
}

func (v *ValidationOutcome) Error() string {
	errs := v.errors
	if len(errs) == 0 {
		return ""
	}

	keys, i := make([]string, len(errs)), 0
	for key := range errs {
		keys[i] = key
		i++
	}
	sort.Strings(keys)

	var s strings.Builder
	for i, key := range keys {
		if i > 0 {
			s.WriteString("; ")
		}
		_, _ = fmt.Fprintf(&s, "%v: ", key)
		for ii, err := range errs[key] {
			if ii > 0 {
				s.WriteString(", ")
			}
			if vr, ok := err.(*ValidationOutcome); ok {
				_, _ = fmt.Fprintf(&s, "(%v)", vr.Error())
			} else {
				s.WriteString(err.Error())
			}
		}
	}
	return s.String()
}

func (v *ValidationOutcome) nestedOutcome(
	key             string,
	createIfMissing bool,
) *ValidationOutcome {
	if v.errors == nil {
		if createIfMissing {
			outcome := &ValidationOutcome{}
			v.errors = map[string][]error{key: {outcome}}
			return outcome
		}
		return nil
	}
	keyErrors, found := v.errors[key]
	if found {
		for _, err := range keyErrors {
			if vr, ok := err.(*ValidationOutcome); ok {
				return vr
			}
		}
	}
	outcome := &ValidationOutcome{}
	v.errors[key] = append(keyErrors, outcome)
	return outcome
}

func (v *ValidationOutcome) parsePath(
	path            string,
	createIfMissing bool,
) (parent *ValidationOutcome, key string) {
	parent = v
	for parent != nil {
		if index, rest := v.parseIndexer(path); len(index) > 0 {
			if len(rest) == 0 {
				return parent, index
			}
			parent = parent.nestedOutcome(index, createIfMissing)
		} else {
			dot  := strings.IndexRune(path, '.')
			open := strings.IndexRune(path, '[')
			if dot > 0 || open > 0 {
				var rest string
				if dot > 0 && (open < 0 || dot < open) {
					rest, path = path[(dot + 1):], path[0:dot]
				} else {
					rest, path = path[open:], path[0:open]
				}
				if len(rest) == 0 {
					return parent, path
				}
				parent, path = parent.nestedOutcome(path, createIfMissing), rest
			} else {
				return parent, path
			}
		}
	}
	return nil, path
}

func (v *ValidationOutcome) parseIndexer(
	path string,
) (index string, rest string) {
	if start := strings.IndexRune(path, '['); start != 0 {
		return "", path
	} else if end := strings.IndexRune(path, ']'); end <= start {
		panic("invalid property indexer")
	} else {
		if index := path[1:end]; len(index) == 0 {
			panic("missing property index")
		} else {
			return index, strings.Trim(path[end + 1:], ".")
		}
	}
}

// validateFilter validates business rules.
type validateFilter struct {}

func (v validateFilter) Order() int {
	return FilterStageValidation
}

func (v validateFilter) Next(
	next     Next,
	context  HandleContext,
	provider FilterProvider,
)  (result []any, err error) {
	if vp, ok := provider.(*ValidateProvider); ok {
		composer := context.Composer()
		outcomeIn, errIn := Validate(composer, context.Callback())
		if errIn != nil {
			return nil, errIn
		}
		if !outcomeIn.Valid() {
			return nil, outcomeIn
		}
		result, err = next.Filter()
		if vp.validateResult && len(result) > 0 && !IsNil(result[0]) {
			outcomeOut, errOut := Validate(composer, result[0])
			if errOut != nil {
				return nil, errOut
			}
			if !outcomeOut.Valid() {
				return nil, outcomeOut
			}
		}
		return result, err
	}
	return next.Abort()
}

// ValidateProvider is a FilterProvider for validation.
type ValidateProvider struct {
	validateResult bool
}

func (v *ValidateProvider) InitWithTag(tag reflect.StructTag) error {
	if validate, ok := tag.Lookup("validate"); ok {
		v.validateResult = validate == "result"
	}
	return nil
}

func (v *ValidateProvider) Required() bool {
	return false
}

func (v *ValidateProvider) Filters(
	binding  Binding,
	callback any,
	composer Handler,
) ([]Filter, error) {
	return _validateFilter, nil
}

func NewValidateProvider(withResult bool) *ValidateProvider {
	return &ValidateProvider{withResult}
}

// ValidatesBuilder builds Validates callbacks.
type ValidatesBuilder struct {
	CallbackBuilder
	target any
	scopes []any
}

func (b *ValidatesBuilder) Target(
	target any,
) *ValidatesBuilder {
	if IsNil(target) {
		panic("target cannot be nil")
	}
	b.target = target
	return b
}

func (b *ValidatesBuilder) WithScopes(
	scopes ... any,
) *ValidatesBuilder {
	b.scopes = scopes
	return b
}

func (b *ValidatesBuilder) NewValidates() *Validates {
	validates := &Validates{
		CallbackBase: b.CallbackBase(),
		target:       b.target,
	}
	if scopes := b.scopes; len(scopes) > 0 {
		validates.scopes   = scopes
		validates.metadata = BindingMetadata{}
		scopeMap := make(map[any]struct{})
		for _, scope := range scopes {
			scopeMap[scope] = struct{}{}
		}
		(&Scope{scopes: scopeMap}).Require(&validates.metadata)
	}
	return validates
}

func Validate(
	handler Handler,
	target  any,
	scopes  ... any,
) (*ValidationOutcome, error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	var builder ValidatesBuilder
	builder.Target(target).WithMany()
	if len(scopes) > 0 {
		builder.WithScopes(scopes...)
	}
	validates := builder.NewValidates()
	if result := handler.Handle(validates, true, nil); result.IsError() {
		return nil, result.Error()
	} else if !result.handled {
		return nil, NotHandledError{validates}
	}
	outcome := validates.Outcome()
	if v, ok := target.(interface {
		SetValidationOutcome(*ValidationOutcome)
	}); ok {
		v.SetValidationOutcome(outcome)
	}
	return outcome, nil
}

var (
	_validatesPolicy Policy = &ContravariantPolicy{}
	_validateFilter = []Filter{validateFilter{}}
)
