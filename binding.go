package miruken

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

type (
	// Binding abstracts a Callback handler.
	Binding interface {
		Filtered
		Key()         any
		Strict()      bool
		SkipFilters() bool
		Metadata()    []any
		Invoke(
			ctx HandleContext,
			explicitArgs ... any,
		) (results []any, err error)
	}

	// BindingReducer aggregates Binding results.
	BindingReducer func(
		binding Binding,
		result  HandleResult,
	) (HandleResult, bool)
)

type (
	bindingFlags uint8

	// Strict Binding's do not expand results.
	Strict struct{}

	// Optional marks a dependency not required.
	Optional struct{}

	// SkipFilters skips all non-required filters.
	SkipFilters struct{}
)

const (
	bindingNone bindingFlags = 0
	bindingStrict bindingFlags = 1 << iota
	bindingOptional
	bindingSkipFilters
)

type (
	bindingBuilder interface {
		configure(
			index   int,
			field   reflect.StructField,
			binding any,
		) (bound bool, err error)
	}

	bindingBuilderFunc func (
		index   int,
		field   reflect.StructField,
		binding any,
	) (bound bool, err error)
)

func (b bindingBuilderFunc) configure(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	return b(index, field, binding)
}

func configureBinding(
	source   reflect.Type,
	binding  any,
	builders []bindingBuilder,
) (err error) {
	checkedMetadata := false
	var metadataOwner interface {
		addMetadata(metadata any) error
	}
	for i := 0; i < source.NumField(); i++ {
		bound := false
		field := source.Field(i)
		for _, builder := range builders {
			if b, invalid := builder.configure(i, field, binding); invalid != nil {
				err = multierror.Append(err, invalid)
				break
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
	if err == nil {
		if b, ok := binding.(interface {
			complete() error
		}); ok {
			err = b.complete()
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

func bindOptions(
	index   int,
	field   reflect.StructField,
	binding any,
) (bound bool, err error) {
	typ := field.Type
	if typ == _strictType {
		bound = true
		if b, ok := binding.(interface {
			setStrict(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setStrict(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"bindOptions: strict field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == _optionalType {
		bound = true
		if b, ok := binding.(interface {
			setOptional(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setOptional(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"bindOptions: optional field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	} else if typ == _skipFiltersType {
		bound = true
		if b, ok := binding.(interface {
			setSkipFilters(int, reflect.StructField, bool) error
		}); ok {
			if invalid := b.setSkipFilters(index, field, true); invalid != nil {
				err = multierror.Append(err, fmt.Errorf(
					"bindOptions: skipFilters on field %v (%v) failed: %w",
					field.Name, index, invalid))
			}
		}
	}
	return bound, err
}

var (
	_strictType      = TypeOf[Strict]()
	_optionalType    = TypeOf[Optional]()
	_skipFiltersType = TypeOf[SkipFilters]()
)