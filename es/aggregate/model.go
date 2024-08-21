package aggregate

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/es/internal"
)

// Model captures an aggregate specification.
type Model struct {
	name       string
	id         func(any) uuid.UUID
	setId      func(any, uuid.UUID)
	version    func(any) int
	setVersion func(any, int)
}

func (m *Model) Name() string {
	return m.name
}

func (m *Model) InitWithTag(tag reflect.StructTag) error {
	if aggregate, ok := tag.Lookup("aggregate"); ok {
		_, err := fmt.Sscanf(aggregate, "name=%s", &m.name)
		return err
	}
	return nil
}

func (m *Model) InitWithBinding(binding miruken.Binding) (err error) {
	if m.name == "" {
		m.name = internal.DefaultBindingName(binding)
	}

	typ := binding.LogicalOutputType()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() == reflect.Struct {
		if err = m.extractIdAndVersionFields(typ); err != nil {
			return
		}
	}

	if m.id == nil || m.setId == nil || m.version == nil || m.setVersion == nil {
		m.extractIdAndVersionMethods(typ)
	}

	if m.id == nil {
		err = errors.Join(err, fmt.Errorf("aggregate: missing id field or getter"))
	}

	if m.setId == nil {
		err = errors.Join(err, fmt.Errorf("aggregate: missing id field or setter"))
	}

	if m.version == nil {
		err = errors.Join(err, fmt.Errorf("aggregate: missing version field or getter"))
	}

	if m.setVersion == nil {
		err = errors.Join(err, fmt.Errorf("aggregate: missing version field or setter"))
	}

	return
}

func (m *Model) extractIdAndVersionFields(typ reflect.Type) (err error) {
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
		m.setId = func(a any, id uuid.UUID) {
			val := reflect.ValueOf(a)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			val.Field(idIndex).Set(reflect.ValueOf(id))
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
		m.setVersion = func(a any, version int) {
			val := reflect.ValueOf(a)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			val.Field(versionIndex).SetInt(int64(version))
		}
	}

	return nil
}

func (m *Model) extractIdAndVersionMethods(typ reflect.Type) {
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
		if m.setId == nil &&
			method.Name == "SetId" &&
			method.Type.NumIn() == 2 &&
			method.Type.In(1) == internal.UUIDType {
			f := method.Func
			m.setId = func(a any, id uuid.UUID) {
				f.Call([]reflect.Value{reflect.ValueOf(a), reflect.ValueOf(id)})
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
		if m.setVersion == nil &&
			method.Name == "SetVersion" &&
			method.Type.NumIn() == 2 &&
			method.Type.In(1).Kind() == reflect.Int {
			f := method.Func
			m.setVersion = func(a any, version int) {
				f.Call([]reflect.Value{reflect.ValueOf(a), reflect.ValueOf(version)})
			}
		}

		if m.id != nil && m.setId != nil && m.version != nil || m.setVersion != nil {
			return
		}
	}
}