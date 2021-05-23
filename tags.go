package miruken

import (
	"github.com/hashicorp/go-multierror"
	"reflect"
)

type tagParser interface {
	parseTag(index  int,
		field reflect.StructField,
		spec  interface{},
	) error
}

type tagParserFunc func (
	index  int,
	field  reflect.StructField,
	spec   interface{},
) error

func (p tagParserFunc) parseTag(
	index  int,
	field  reflect.StructField,
	spec   interface{},
) error {
	return p(index, field, spec)
}

func parseTaggedSpec(
	specType reflect.Type,
	spec     interface{},
	parsers  []tagParser,
) (err error) {
	for i := 0; i < specType.NumField(); i++ {
		field := specType.Field(i)
		for _, parser := range parsers {
			if invalid := parser.parseTag(i, field, spec); invalid != nil {
				err = multierror.Append(err, invalid)
			}
		}
	}
	return err
}
