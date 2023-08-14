package stdjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Rican7/conjson/transform"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/maps"
	"io"
	"reflect"
	"strings"
)

type (
	// typeContainer customizes json standard serialization to
	// emit type field information needed to support polymorphism.
	typeContainer struct {
		v        any
		typInfo  string
		trans    []transform.Transformer
		composer miruken.Handler
	}
)

var (
	// KnownTypeFields holds the list of json property names
	// that can contain type discriminators.
	KnownTypeFields = []string{"$type", "@type"}

	// KnownValuesFields holds the list of json property names
	// that can contain values for discriminated arrays.
	KnownValuesFields = []string{"$values", "@values"}
)


func (c *typeContainer) typeInfo() *maps.Format {
	if typeInfo := c.typInfo; len(typeInfo) > 0 {
		return maps.To(typeInfo, nil)
	}
	return api.ToTypeInfo
}

func (c *typeContainer) MarshalJSON() ([]byte, error) {
	v   := c.v
	typ := reflect.TypeOf(v)
	if typ != nil && typ.Kind() == reflect.Slice {
		et  := typ.Elem()
		s   := reflect.ValueOf(v)
		arr := make([]*json.RawMessage, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			var b bytes.Buffer
			writer := io.Writer(&b)
			enc    := json.NewEncoder(writer)
			elem   := s.Index(i).Interface()
			if anyType.AssignableTo(et) || reflect.TypeOf(elem) != et {
				elem = &typeContainer{
					v:        elem,
					typInfo:  c.typInfo,
					trans:    c.trans,
					composer: c.composer,
				}
			}
			if err := enc.Encode(elem); err != nil {
				return nil, fmt.Errorf("can't marshal array index %d: %w", i, err)
			} else {
				raw := json.RawMessage(b.Bytes())
				arr = append(arr, &raw)
			}
		}
		v = arr
	}
	vm := v
	if trans := c.trans; len(trans) > 0 {
		vm = &transformer{v, trans}
	}
	byt, err := json.Marshal(vm)
	if err != nil || len(byt) == 0 {
		return byt, err
	}
	if byt[0] == '{' {
		typeInfo, _, _, err := maps.Out[api.TypeFieldInfo](c.composer, v, c.typeInfo())
		if err != nil {
			return nil, err
		}
		var comma string
		if len(byt) > 1 && byt[1] != '}' {
			comma = ","
		}
		typeProperty := []byte(fmt.Sprintf("%q:%q%s", typeInfo.TypeField, typeInfo.TypeValue, comma))
		byt = append(byt, typeProperty...)
		copy(byt[len(typeProperty)+1:], byt[1:])
		copy(byt[1:], typeProperty)
	} else if byt[0] == '[' {
		typeInfo, _, _, err := maps.Out[api.TypeFieldInfo](c.composer, c.v, c.typeInfo())
		if err != nil {
			return nil, err
		}
		var sb strings.Builder
		sb.WriteString("{\"")
		sb.WriteString(typeInfo.TypeField)
		sb.WriteString("\":\"")
		sb.WriteString(typeInfo.TypeValue)
		sb.WriteString("\",\"")
		sb.WriteString(typeInfo.ValuesField)
		sb.WriteString("\":")
		sb.Write(byt)
		sb.WriteString("}")
		byt = []byte(sb.String())
	}
	return byt, nil
}

func (c *typeContainer) UnmarshalJSON(data []byte) error {
	var fields map[string]*json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		if me, ok := err.(*json.UnmarshalTypeError); ok {
			if me.Value == "array" {
				var raw []*json.RawMessage
				if err = json.Unmarshal(data, &raw); err == nil {
					var arr reflect.Value
					typ := reflect.Indirect(reflect.ValueOf(c.v)).Type()
					if typ.Kind() == reflect.Slice {
						arr = reflect.MakeSlice(typ, len(raw), len(raw))
					} else {
						arr = reflect.ValueOf(make([]any, len(raw), len(raw)))
					}
					for i, elem := range raw {
						r   := bytes.NewReader(*elem)
						dec := json.NewDecoder(r)
						tc  := typeContainer{
							v:        arr.Index(i).Addr().Interface(),  // &arr[0]
							typInfo:  c.typInfo,
							trans:    c.trans,
							composer: c.composer,
						}
						if err := dec.Decode(&tc); err != nil {
							return fmt.Errorf("can't unmarshal array index %d: %w", i, err)
						}
					}
					internal.CopyIndirect(arr.Interface(), c.v)
				}
			} else {
				return json.Unmarshal(data, c.v)
			}
		}
		return err
	}
	var (
		field     string
		typeIdRaw *json.RawMessage
	)
	for _, field = range KnownTypeFields {
		if typeIdRaw = fields[field]; typeIdRaw != nil {
			break
		}
	}
	if typeIdRaw == nil {
		if late, ok := c.v.(*api.Late); ok {
			if err := json.Unmarshal(data, &late.Value); err != nil {
				return err
			} else {
				return nil
			}
		}
		return json.Unmarshal(data, c.v)
	}
	var typeId string
	if err := json.Unmarshal(*typeIdRaw, &typeId); err != nil {
		return err
	} else if len(typeId) == 0 {
		return fmt.Errorf("empty type id for field %q", field)
	} else {
		if v, _, err := creates.Key[any](c.composer, typeId); err != nil {
			return &api.UnknownTypeIdError{TypeId: typeId, Cause: err}
		} else {
			vm := v
			for _, field = range KnownValuesFields {
				if values := fields[field]; values != nil {
					data = *values
					vm   = &typeContainer{
						v:        v,
						typInfo:  c.typInfo,
						trans:    c.trans,
						composer: c.composer,
					}
				}
			}
			if trans := c.trans; len(trans) > 0 {
				vm = &transformer{vm, trans}
			}
			if err := json.Unmarshal(data, vm); err != nil {
				return err
			} else {
				if late, ok := c.v.(*api.Late); ok {
					late.Value = v
				} else {
					internal.CopyIndirect(v, c.v)
				}
			}
		}
	}
	return nil
}

var anyType = internal.TypeOf[any]()