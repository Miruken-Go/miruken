package internal

import (
	"path"
	"reflect"

	"github.com/miruken-go/miruken/internal"
)

func TypeName(v any) string {
	if internal.IsNil(v) {
		return ""
	}

	typ, ok := v.(reflect.Type)
	if !ok {
		typ = reflect.TypeOf(v)
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	typeName := typ.Name()
	if packagePath := typ.PkgPath(); packagePath == "" {
		return typeName
	} else {
		return path.Base(packagePath) + "." + typeName
	}
}
