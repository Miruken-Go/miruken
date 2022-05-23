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

	// BindingMetadataFactory create new Binding metadata.
	BindingMetadataFactory interface {
		Build(reflect.StructField) any
	}
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
		if !bound {
			if b, ok := binding.(interface {
				unknown(int, reflect.StructField) error
			}); ok {
				if invalid := b.unknown(i, field); invalid != nil {
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