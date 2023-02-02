package jsonstd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Rican7/conjson"
	"github.com/Rican7/conjson/transform"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	jsonapi "github.com/miruken-go/miruken/api/json"
	"github.com/miruken-go/miruken/maps"
	"io"
	"reflect"
)

type (
	// Mapper formats to and from json using encoding/json.
	Mapper struct{}

	// Options provide options for controlling json encoding.
	Options struct {
		Prefix       string
		Indent       string
		EscapeHTML   miruken.Option[bool]
		Transformers []transform.Transformer
	}
)

func (m *Mapper) ToJson(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, maps *maps.It,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	ctx miruken.HandleContext,
) (js string, err error) {
	var data []byte
	src := maps.Source()
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		src = &typeContainer{
			v:        src,
			typInfo:  apiOptions.TypeInfoFormat,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
	} else if trans := options.Transformers; len(trans) > 0 {
		src = &transformer{src, trans}
	}
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(src, prefix, indent)
	} else {
		data, err = json.Marshal(src)
	}
	return string(data), err
}

func (m *Mapper) ToJsonStream(
	_*struct{
		maps.It
		maps.Format `to:"application/json"`
	  }, maps *maps.It,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	ctx miruken.HandleContext,
) (stream io.Writer, err error) {
	if writer, ok := maps.Target().(*io.Writer); ok && !miruken.IsNil(writer) {
		enc := json.NewEncoder(*writer)
		if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
			enc.SetIndent(prefix, indent)
		}
		if escapeHTML := options.EscapeHTML; escapeHTML.Set() {
			enc.SetEscapeHTML(escapeHTML.Value())
		}
		src := maps.Source()
		if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
			src = &typeContainer{
				v:        src,
				typInfo:  apiOptions.TypeInfoFormat,
				trans:    options.Transformers,
				composer: ctx.Composer()}
		} else if trans := options.Transformers; len(trans) > 0 {
			src = &transformer{src, trans}
		}
		err    = enc.Encode(src)
		stream = *writer
	}
	return stream, err
}

func (m *Mapper) FromJson(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, jsonString string,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	maps *maps.It,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		tc := typeContainer{
			v:        target,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
		err := json.Unmarshal([]byte(jsonString), &tc)
		return target, err
	} else if trans := options.Transformers; len(trans) > 0 {
		t := transformer{target, trans}
		target = &t
	}
	err := json.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *Mapper) FromJsonStream(
	_*struct{
		maps.It
		maps.Format `from:"application/json"`
	  }, stream io.Reader,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options Options,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, apiOptions api.Options,
	maps *maps.It,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	dec    := json.NewDecoder(stream)
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		tc := typeContainer{
			v:        target,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
		err := dec.Decode(&tc)
		return target, err
	} else if trans := options.Transformers; len(trans) > 0 {
		t := transformer{target, trans}
		target = &t
	}
	err := dec.Decode(target)
	return target, err
}

// typeContainer is a helper type used to emit type field
// information for polymorphic serialization/deserialization.
type typeContainer struct {
	v        any
	typInfo  string
	trans    []transform.Transformer
	composer miruken.Handler
}

func (c *typeContainer) TypeInfo() *maps.Format {
	if typeInfo := c.typInfo; len(typeInfo) > 0 {
		return maps.To(typeInfo)
	}
	return api.ToTypeInfo
}

func (c *typeContainer) MarshalJSON() ([]byte, error) {
	v   := c.v
	typ := reflect.TypeOf(v)
	if typ != nil && typ.Kind() == reflect.Slice {
		s   := reflect.ValueOf(v)
		arr := make([]*json.RawMessage, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			var b bytes.Buffer
			writer := io.Writer(&b)
			enc    := json.NewEncoder(writer)
			elem   := typeContainer{
				v:        s.Index(i).Interface(),
				typInfo:  c.typInfo,
				trans:    c.trans,
				composer: c.composer,
			}
			if err := enc.Encode(&elem); err != nil {
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
	if byt, err := json.Marshal(vm); err != nil {
		return nil, err
	} else if len(byt) > 0 && byt[0] == '{' {
		typeInfo, _, err := maps.Map[api.TypeFieldInfo](c.composer, v, c.TypeInfo())
		if err != nil {
			return nil, err
		}
		var comma string
		if len(byt) > 1 && byt[1] != '}' {
			comma = ","
		}
		typeProperty := []byte(fmt.Sprintf("\"%v\":\"%v\"%s", typeInfo.Field, typeInfo.Value, comma))
		byt = append(byt, typeProperty...)
		copy(byt[len(typeProperty)+1:], byt[1:])
		copy(byt[1:], typeProperty)
		return byt, nil
	} else {
		return byt, nil
	}
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
							v:       &target,
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
	for _, field = range jsonapi.KnownTypeFields {
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


// transformer applies transformations to json serialization.
type transformer struct {
	v     any
	trans []transform.Transformer
}

func (t *transformer) MarshalJSON() ([]byte, error) {
	conventions := conjson.NewMarshaler(t.v, t.trans...)
	return json.Marshal(conventions)
}

func (t *transformer) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, conjson.NewUnmarshaler(t.v, t.trans...))
}

var (
	// CamelCase directs the json encoding of keys to use camelcase notation.
	CamelCase = miruken.Options(Options{
		Transformers: []transform.Transformer{
			transform.OnlyForDirection(
				transform.Marshal,
				transform.CamelCaseKeys(false)),
		},
	})
)