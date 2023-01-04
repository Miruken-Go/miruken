package validate

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/slices"
	"sort"
	"strings"
)

type (
	// Outcome captures structured validation errors.
	Outcome struct {
		errors map[string][]error
	}

	// ApiOutcome transports Outcome information over an api.
	ApiOutcome struct {
		PropertyName string
		Errors       []string
		Nested       []ApiOutcome
	}

	// ApiMapping maps validation errors for api transport.
	ApiMapping struct {}
)


// Outcome

func (v *Outcome) Valid() bool {
	return v.errors == nil
}

func (v *Outcome) Fields() []string {
	var keys []string
	if errs := v.errors; len(errs) > 0 {
		keys = maps.Keys(v.errors)
	}
	return keys
}

func (v *Outcome) AddError(
	path string,
	err  error,
) {
	if err == nil {
		panic("err cannot be nil")
	}
	if _, ok := err.(*Outcome); ok {
		panic("cannot add path Outcome directly")
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

func (v *Outcome) FieldErrors(
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

func (v *Outcome) Path(
	path string,
) *Outcome {
	if parent, key := v.parsePath(path, false); parent == v {
		return v.childPath(key, false)
	} else if parent != nil {
		return parent.Path(key)
	}
	return nil
}

func (v *Outcome) RequirePath(
	path string,
) *Outcome {
	if parent, key := v.parsePath(path, true); parent == v {
		return v.childPath(key, true)
	} else {
		return parent.RequirePath(key)
	}
}

func (v *Outcome) Error() string {
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
			if vr, ok := err.(*Outcome); ok {
				_, _ = fmt.Fprintf(&s, "(%v)", vr.Error())
			} else {
				s.WriteString(err.Error())
			}
		}
	}
	return s.String()
}

func (v *Outcome) childPath(
	key     string,
	require bool,
) *Outcome {
	if v.errors == nil {
		if require {
			outcome := &Outcome{}
			v.errors = map[string][]error{key: {outcome}}
			return outcome
		}
		return nil
	}
	keyErrors, found := v.errors[key]
	if found {
		for _, err := range keyErrors {
			if vr, ok := err.(*Outcome); ok {
				return vr
			}
		}
	}
	if require {
		outcome := &Outcome{}
		v.errors[key] = append(keyErrors, outcome)
		return outcome
	}
	return nil
}

func (v *Outcome) parsePath(
	path    string,
	require bool,
) (parent *Outcome, key string) {
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

func (v *Outcome) parseIndexer(
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


// ApiMapping

func (m *ApiMapping) ForApi(
	_*struct{
		miruken.Maps
		miruken.Format `to:"api:error"`
	  }, outcome *Outcome,
) (ao []ApiOutcome) {
	if ao = buildApiOutcome(outcome); ao == nil {
		ao = []ApiOutcome{}
	}
	return
}

func (m *ApiMapping) FromApi(
	_*struct{
		miruken.Maps
		miruken.Format `from:"api:error"`
	}, apiOutcome []*ApiOutcome,
) (*Outcome, error) {
	return buildOutcome(slices.Map[*ApiOutcome, ApiOutcome](apiOutcome,
		func(ao *ApiOutcome) ApiOutcome {
			return *ao
		})), nil
}

func (m *ApiMapping) New(
	_*struct{
		miruken.Creates `key:"validate.ApiOutcome"`
	  },
) *ApiOutcome {
	return new(ApiOutcome)
}

func buildApiOutcome(outcome *Outcome) []ApiOutcome {
	return slices.Map[string, ApiOutcome](
		outcome.Fields(),
		func(field string) ApiOutcome {
			var messages []string
			var children []ApiOutcome
			for _, err := range outcome.FieldErrors(field) {
				if child, ok := err.(*Outcome); ok {
					children = append(children, buildApiOutcome(child)...)
				} else {
					messages = append(messages, err.Error())
				}
			}
			return ApiOutcome{field, messages, children}
		})
}

func buildOutcome(apiOutcome []ApiOutcome) *Outcome {
	outcome := &Outcome{}
	for _, ao := range apiOutcome {
		field := ao.PropertyName
		if failures := ao.Errors; len(failures) > 0 {
			for _, msg := range failures {
				outcome.AddError(field, errors.New(msg))
			}
		}
		if nested := ao.Nested; len(nested) > 0 {
			outcome.AddError(field, buildOutcome(nested))
		}
	}
	return outcome
}