package jsonstd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Rican7/conjson/transform"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/maps"
	"io"
	"reflect"
	"strings"
)

type (
	// apiMessage is the json specific stub for the generic api.Message.
	apiMessage struct {
		Payload json.RawMessage `json:"payload"`
	}

	// apiMessageMapper maps between the generic api.Message and the json
	// specific apiMessage stub.
	apiMessageMapper struct {}

	// typeContainer intercepts json serialization to emit type field
	// information needed to support polymorphism.
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


// apiMessageMapper

func (m *apiMessageMapper) Encode(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, msg api.Message,
	it *maps.It,
	ctx miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := it.Target().(*io.Writer); ok {
		var sur apiMessage
		if payload := msg.Payload; payload != nil {
			pj, _, err := maps.Map[string](ctx.Composer(), msg.Payload, api.ToJson)
			if err != nil {
				return nil, err
			}
			sur.Payload = json.RawMessage(pj)
		}
		enc := json.NewEncoder(*writer)
		if err := enc.Encode(sur); err == nil {
			return *writer, err
		}
	}
	return nil, nil
}

func (m *apiMessageMapper) Decode(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, stream io.Reader,
	ctx miruken.HandleContext,
) (api.Message, error) {
	var sur apiMessage
	dec := json.NewDecoder(stream)
	if err := dec.Decode(&sur); err != nil {
		return api.Message{}, err
	}
	payload, _, err := maps.Map[any](ctx.Composer(), string(sur.Payload), api.FromJson)
	if sur, ok := payload.(api.Surrogate); ok {
		payload = sur.Original()
	}
	return api.Message{Payload: payload}, err
}


// typeContainer

func (c *typeContainer) TypeInfo() *maps.Format {
	if typeInfo := c.typInfo; len(typeInfo) > 0 {
		return maps.To(typeInfo)
	}
	return api.ToTypeInfo
}

func (c *typeContainer) MarshalJSON() ([]byte, error) {
	v   := c.v
	typ := reflect.TypeOf(v)
	var elemTyp reflect.Type
	if typ != nil && typ.Kind() == reflect.Slice {
		elemTyp = typ.Elem()
		s   := reflect.ValueOf(v)
		arr := make([]*json.RawMessage, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			var b bytes.Buffer
			writer := io.Writer(&b)
			enc    := json.NewEncoder(writer)
			elem   := s.Index(i).Interface()
			if reflect.TypeOf(elem) != elemTyp {
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
		typeInfo, _, err := maps.Map[api.TypeFieldInfo](c.composer, v, c.TypeInfo())
		if err != nil {
			return nil, err
		}
		var comma string
		if len(byt) > 1 && byt[1] != '}' {
			comma = ","
		}
		typeProperty := []byte(fmt.Sprintf("\"%v\":\"%v\"%s",
			typeInfo.TypeField, typeInfo.TypeValue, comma))
		byt = append(byt, typeProperty...)
		copy(byt[len(typeProperty)+1:], byt[1:])
		copy(byt[1:], typeProperty)
	} else if byt[0] == '[' && elemTyp != nil && !anyType.AssignableTo(elemTyp) {
		typeInfo, _, err := maps.Map[api.TypeFieldInfo](c.composer, c.v, c.TypeInfo())
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
					var elemTyp reflect.Type
					typ := reflect.Indirect(reflect.ValueOf(c.v)).Type()
					if typ.Kind() == reflect.Slice {
						arr = reflect.MakeSlice(typ, 0, len(raw))
						elemTyp = arr.Type().Elem()
					} else {
						arr = reflect.ValueOf(make([]any, 0, len(raw)))
					}
					for i, elem := range raw {
						var target any
						r   := bytes.NewReader(*elem)
						dec := json.NewDecoder(r)
						tc  := typeContainer{
							v:        &target,
							typInfo:  c.typInfo,
							trans:    c.trans,
							composer: c.composer,
						}
						if err := dec.Decode(&tc); err != nil {
							return fmt.Errorf("can't unmarshal array index %d: %w", i, err)
						} else {
							v := reflect.ValueOf(target)
							if elemTyp != nil {
								v = v.Convert(elemTyp)
							}
							arr = reflect.Append(arr, v)
						}
					}
					miruken.CopyIndirect(arr.Interface(), c.v)
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
		if late, ok := c.v.(*miruken.Late); ok {
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
		return fmt.Errorf("empty type id for field \"%s\"", field)
	} else {
		if v, _, err := miruken.CreateKey[any](c.composer, typeId); err != nil {
			return &api.UnknownTypeIdError{TypeId: typeId, Reason: err}
		} else {
			vm := v
			if trans := c.trans; len(trans) > 0 {
				vm = &transformer{v, trans}
			}
			for _, field = range KnownValuesFields {
				if values := fields[field]; values != nil {
					data = *values
				}
			}
			if err := json.Unmarshal(data, vm); err != nil {
				return err
			} else {
				if late, ok := c.v.(*miruken.Late); ok {
					late.Value = v
				} else {
					miruken.CopyIndirect(v, c.v)
				}
			}
		}
	}
	return nil
}

var anyType = miruken.TypeOf[any]()