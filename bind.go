package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// Binding connects a Callback to a handler.
	Binding interface {
		Filtered
		Key()         any
		Strict()      bool
		SkipFilters() bool
		Metadata()    []any
		Invoke(
			ctx HandleContext,
			initArgs ...any,
		) ([]any, *promise.Promise[[]any], error)
	}

	// BindingBase implements common binding contract.
	BindingBase struct {
		FilteredScope
		flags    bindingFlags
		metadata []any
	}

	// BindingReducer aggregates Binding results.
	BindingReducer func(
		binding Binding,
		result  HandleResult,
	) (HandleResult, bool)

	// Late is a container for late Binding results.
	Late struct {
		Value any
	}

	BindingParser interface {
		parse(
			index   int,
			field   reflect.StructField,
			binding any,
		) (bound bool, err error)
	}

	BindingParserFunc func (
		index   int,
		field   reflect.StructField,
		binding any,
	) (bound bool, err error)
)


type (
	bindingFlags uint8

	// Strict Binding's do not expand results.
	Strict struct{}

	// Optional marks a dependency not required.
	Optional struct{}

	// SkipFilters skips all non-required filters.
	SkipFilters struct{}

	// BindingGroup marks bindings that aggregate
	// one or more binding metadata.
	BindingGroup struct {}
)


const (
	bindingNone bindingFlags = 0
	bindingStrict bindingFlags = 1 << iota
	bindingOptional
	bindingSkipFilters
	bindingPromise
)


// BindingGroup

func (BindingGroup) DefinesBindingGroup() {}


// BindingBase

func (b *BindingBase) Strict() bool {
	return b.flags & bindingStrict == bindingStrict
}

func (b *BindingBase) SkipFilters() bool {
	return b.flags & bindingSkipFilters == bindingSkipFilters
}

func (b *BindingBase) Metadata() []any {
	return b.metadata
}


// BindingParserFunc

func (b BindingParserFunc) parse(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	return b(index, field, binding)
}


func parseBinding(
	source  reflect.Type,
	binding any,
	parsers []BindingParser,
) (err error) {
	if err = parseStructBinding(source, binding, parsers); err == nil {
		if b, ok := binding.(interface {
			complete() error
		}); ok {
			err = b.complete()
		}
	}
	return
}

func parseStructBinding(
	typ     reflect.Type,
	binding any,
	parsers []BindingParser,
) (err error) {
	checkedMetadata := false
	var metadataOwner interface {
		addMetadata(metadata any) error
	}
	NextField:
	for i := 0; i < typ.NumField(); i++ {
		bound := false
		field := typ.Field(i)
		fieldType := field.Type
		if fieldType == bindingGroupType {
			continue
		}
		if fieldType.Kind() == reflect.Struct && fieldType.Implements(definesBindingGroup) {
			if invalid := parseStructBinding(fieldType, binding, parsers); invalid != nil {
				err = multierror.Append(err, invalid)
			}
			continue
		}
		for _, parser := range parsers {
			if b, invalid := parser.parse(i, field, binding); invalid != nil {
				err = multierror.Append(err, invalid)
				continue NextField
			} else if b {
				bound = true
				break
			}
		}
		if !bound && (metadataOwner != nil || !checkedMetadata) {
			if !checkedMetadata {
				checkedMetadata = true
				metadataOwner, _ = binding.(interface {
					addMetadata(metadata any) error
				})
			}
			if metadataOwner != nil {
				if invalid := addMetadata(field.Type, field.Tag, metadataOwner); invalid != nil {
					err = multierror.Append(err, invalid)
				}
			}
		}
	}
	return err
}

func addMetadata(
	typ   reflect.Type,
	tag   reflect.StructTag,
	owner interface {
		addMetadata(metadata any) error
	},
) error {
	writeable := typ.Kind() == reflect.Ptr
	if !writeable {
		typ = reflect.PtrTo(typ)
	}
	if metadata, err := newWithTag(typ, tag); metadata != nil && err == nil {
		if !writeable {
			metadata = reflect.Indirect(reflect.ValueOf(metadata)).Interface()
		}
		return owner.addMetadata(metadata)
	} else {
		return err
	}
}

func parseOptions(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if typ == strictType {
		bound = true
		if b, ok := binding.(interface {
			setStrict(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setStrict(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: strict field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == optionalType {
		bound = true
		if b, ok := binding.(interface {
			setOptional(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setOptional(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: optional field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == skipFiltersType {
		bound = true
		if b, ok := binding.(interface {
			setSkipFilters(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setSkipFilters(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"parseOptions: skipFilters on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	}
	return bound, err
}

var (
	strictType          = TypeOf[Strict]()
	optionalType        = TypeOf[Optional]()
	skipFiltersType     = TypeOf[SkipFilters]()
	bindingGroupType    = TypeOf[BindingGroup]()
	definesBindingGroup = TypeOf[interface{ DefinesBindingGroup() }]()
)