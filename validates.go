package miruken

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"sort"
	"strings"
)

// Validates callbacks contravariantly.
type Validates struct {
	CallbackBase
	source   any
	groups   []any
	outcome  ValidationOutcome
	metadata BindingMetadata
}

func (v *Validates) Source() any {
	return v.source
}

func (v *Validates) Groups() []any {
	return v.groups
}

func (v *Validates) InGroup(group any) bool {
	if len(v.groups) == 0 {
		return false
	}
	for _, grp := range v.groups {
		if grp == group {
			return true
		}
	}
	return false
}

func (v *Validates) Outcome() *ValidationOutcome {
	return &v.outcome
}

func (v *Validates) Key() any {
	return reflect.TypeOf(v.source)
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
	return DispatchPolicy(handler, v, greedy, composer)
}

type Group struct {
	groups map[any]Void
}

func (g *Group) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		g.groups = make(map[any]Void)
		if group := strings.TrimSpace(name); len(group) > 0 {
			g.groups[group] = Void{}
		}
	}
	if len(g.groups) == 0 {
		return errors.New("the Group constraint requires a non-empty `name:group` tag")
	}
	return nil
}

func (g *Group) Merge(constraint BindingConstraint) bool {
	if group, ok := constraint.(*Group); ok {
		for grp := range group.groups {
			g.groups[grp] = Void{}
		}
		return true
	}
	return false
}

func (g *Group) Require(metadata *BindingMetadata) {
	if groups := g.groups; len(groups) > 0 {
		gs, i := make([]any, len(groups)), 0
		for grp := range groups {
			gs[i] = grp
			i++
		}
		metadata.Set(_groupType, gs)
	}
}

func (g *Group) Matches(metadata *BindingMetadata) bool {
	if m, ok := metadata.Get(_groupType); ok {
		if _, all := g.groups[_anyGroup]; all {
			return true
		}
		if groups, ok := m.([]any); ok {
			for _, group := range groups {
				if group == _anyGroup {
					return true
				}
				if _, found := g.groups[group]; found {
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

func (v *ValidationOutcome) Fields() []string {
	var keys []string
	if errs := v.errors; len(errs) > 0 {
		keys = maps.Keys(v.errors)
	}
	return keys
}

func (v *ValidationOutcome) AddError(
	path string,
	err  error,
) {
	if err == nil {
		panic("err cannot be nil")
	}
	if _, ok := err.(*ValidationOutcome); ok {
		panic("cannot add path ValidationOutcome directly")
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

func (v *ValidationOutcome) FieldErrors(
	path string,
) []error {
	if parent, key := v.parsePath(path, true); parent == nil {
		return nil
	} else if parent != v {
		return parent.FieldErrors(key)
	}
	if v.errors != nil {
		if errs, found := v.errors[path]; found {
			return errs
		}
	}
	return nil
}

func (v *ValidationOutcome) Path(
	path string,
) *ValidationOutcome {
	if parent, key := v.parsePath(path, false); parent == v {
		return v.childPath(key, false)
	} else if parent != nil {
		return parent.Path(key)
	}
	return nil
}

func (v *ValidationOutcome) RequirePath(
	path string,
) *ValidationOutcome {
	if parent, key := v.parsePath(path, true); parent == v {
		return v.childPath(key, true)
	} else {
		return parent.RequirePath(key)
	}
}

func (v *ValidationOutcome) Error() string {
	errs := v.errors
	if len(errs) == 0 {
		return ""
	}

	keys := maps.Keys(errs)
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

func (v *ValidationOutcome) childPath(
	key     string,
	require bool,
) *ValidationOutcome {
	if v.errors == nil {
		if require {
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
	if require {
		outcome := &ValidationOutcome{}
		v.errors[key] = append(keyErrors, outcome)
		return outcome
	}
	return nil
}

func (v *ValidationOutcome) parsePath(
	path    string,
	require bool,
) (parent *ValidationOutcome, key string) {
	parent = v
	for parent != nil {
		if index, rest := v.parseIndexer(path); len(index) > 0 {
			if len(rest) == 0 {
				return parent, index
			}
			parent, path = parent.childPath(index, require), rest
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
				parent, path = parent.childPath(path, require), rest
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
	ctx      HandleContext,
	provider FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if _, ok := provider.(*ValidateProvider); ok {
		callback := ctx.Callback()
		composer := ctx.Composer()
		outcomeIn, poi, errIn := Validate(composer, callback.Source())
		if errIn != nil {
			return nil, nil, errIn
		}
		if poi == nil {
			if !outcomeIn.Valid() {
				return nil, nil, outcomeIn
			}
			return next.Pipe()
		}
		return nil, promise.Then(poi, func(outIn *ValidationOutcome) []any {
			if !outIn.Valid() {
				panic(outIn)
			}
			return next.PipeAwait()
		}), nil
	}
	// todo: validate result
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
	target any
	groups []any
}

func (b *ValidatesBuilder) Target(
	target any,
) *ValidatesBuilder {
	if IsNil(target) {
		panic("source cannot be nil")
	}
	b.target = target
	return b
}

func (b *ValidatesBuilder) WithGroups(
	groups ... any,
) *ValidatesBuilder {
	b.groups = groups
	return b
}

func (b *ValidatesBuilder) NewValidates() *Validates {
	validates := &Validates{
		source: b.target,
	}
	if groups := b.groups; len(groups) > 0 {
		validates.groups = groups
		validates.metadata = BindingMetadata{}
		groupMap := make(map[any]Void)
		for _, group := range groups {
			groupMap[group] = Void{}
		}
		(&Group{groups: groupMap}).Require(&validates.metadata)
	}
	return validates
}

// Validate initiates validation of the source.
func Validate(
	handler Handler,
	target  any,
	groups ... any,
) (o *ValidationOutcome, po *promise.Promise[*ValidationOutcome], err error) {
	if handler == nil {
		panic("handler cannot be nil")
	}
	var builder ValidatesBuilder
	builder.Target(target)
	if len(groups) > 0 {
		builder.WithGroups(groups...)
	}
	validates := builder.NewValidates()
	if result := handler.Handle(validates, true, nil); result.IsError() {
		err = result.Error()
	} else if !result.handled {
		err = NotHandledError{validates}
	} else if _, pv := validates.Result(false); pv == nil {
		o = validates.Outcome()
		setTargetValidationOutcome(target, o)
	} else {
		po = promise.Then(pv, func(any) *ValidationOutcome {
			outcome := validates.Outcome()
			setTargetValidationOutcome(target, outcome)
			return outcome
		})
	}
	return
}

func setTargetValidationOutcome(
	target  any,
	outcome *ValidationOutcome,
) {
	if v, ok := target.(interface {
		SetValidationOutcome(*ValidationOutcome)
	}); ok {
		v.SetValidationOutcome(outcome)
	}
}

// ValidationInstaller enables validation support.
type ValidationInstaller struct {
	results bool
}

func (v *ValidationInstaller) ValidateResults() {
	v.results = true
}

func (v *ValidationInstaller) Install(setup *SetupBuilder) error {
	if setup.CanInstall(&_validationTag) {
		setup.AddFilters(NewValidateProvider(v.results))
	}
	return nil
}

func ValidateResults(installer *ValidationInstaller) {
	installer.ValidateResults()
}

func WithValidation(
	config ... func(installer *ValidationInstaller),
) Feature {
	installer := &ValidationInstaller{}
	for _, configure := range config {
		if configure != nil {
			configure(installer)
		}
	}
	return installer
}

var (
	_validatesPolicy Policy = &ContravariantPolicy{}
	_validateFilter         = []Filter{validateFilter{}}
	_groupType              = TypeOf[*Group]()
	_anyGroup               = "*"
	_validationTag byte
)
