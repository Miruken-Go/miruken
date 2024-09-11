package command

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken/es/internal"
)

// Model captures a command specification.
type Model struct {
	typ        reflect.Type
	meta 	   *Metadata
	id         func(any) uuid.UUID
	version    func(any) int
}


func (m *Model) Name() *Metadata {
	return m.meta
}

func (m *Model) Type() reflect.Type {
	return m.typ
}


// NewModel creates a command model for type and name.
func NewModel(
	typ  reflect.Type,
	meta *Metadata,
) (*Model, error) {
	if typ == nil {
		panic("command: typ cannot be nil")
	}

	if meta.Name() == "" {
		*meta = Metadata(internal.DefaultTypeName(typ))
	}
	return &Model{
		typ:  typ,
		meta: meta,
	}, nil
}


func extractCommandIdAndVersionFields(m *Model, typ reflect.Type) (err error) {
	idIndex := -1
	versionIndex := -1

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("aggregate")

		if tag == "id" {
			if idIndex >= 0 {
				err = errors.Join(err, fmt.Errorf(
					"aggregate: invalid field %q: id already assigned to %q",
					field.Name, typ.Field(idIndex).Name))
			} else {
				idIndex = i
			}
		}
		if tag == "version" {
			if versionIndex >= 0 {
				err = errors.Join(err, fmt.Errorf(
					"aggregate: invalid field %q: version already assigned to %q",
					field.Name, typ.Field(versionIndex).Name))
			} else {
				versionIndex = i
			}
		}
	}

	if idIndex < 0 {
		if field, ok := typ.FieldByName("Id"); ok && field.Type == internal.UUIDType {
			idIndex = field.Index[0]
		}
	}

	if idIndex < 0 {
		scopedId := typ.Name() + "Id"
		if field, ok := typ.FieldByName(scopedId); ok && field.Type == internal.UUIDType {
			idIndex = field.Index[0]
		}
	}

	if versionIndex < 0 {
		if field, ok := typ.FieldByName("Version"); ok && field.Type.Kind() == reflect.Int {
			versionIndex = field.Index[0]
		}
	}

	if idIndex >= 0 {
		m.id = func(a any) uuid.UUID {
			val := reflect.ValueOf(a)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			return val.Field(idIndex).Interface().(uuid.UUID)
		}
	}

	if versionIndex >= 0 {
		m.version = func(a any) int {
			val := reflect.ValueOf(a)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			return int(val.Field(idIndex).Int())
		}
	}

	return nil
}

func extractCommandIdAndVersionMethods(m *Model, typ reflect.Type) {
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		if m.id == nil &&
			method.Name == "Id" &&
			method.Type.NumOut() == 1 &&
			method.Type.Out(0) == internal.UUIDType {
			f := method.Func
			m.id = func(a any) uuid.UUID {
				result := f.Call([]reflect.Value{reflect.ValueOf(a)})
				return result[0].Interface().(uuid.UUID)
			}
		}
		if m.version == nil &&
			method.Name == "Version" &&
			method.Type.NumOut() == 1 &&
			method.Type.Out(0).Kind() == reflect.Int {
			f := method.Func
			m.version = func(a any) int {
				result := f.Call([]reflect.Value{reflect.ValueOf(a)})
				return int(result[0].Int())
			}
		}

		if m.id != nil && m.version != nil {
			return
		}
	}
}