package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/api/json/conjson"
	"github.com/miruken-go/miruken/api/json/conjson/transform"
	"io"
	"reflect"
)

type (
	// StdMapper formats to and from json using encoding/json.
	StdMapper struct{}

	// StdOptions provide options for controlling json encoding.
	StdOptions struct {
		Prefix       string
		Indent       string
		EscapeHTML   miruken.Option[bool]
		Transformers []transform.Transformer
	}
)

// StdMapper

func (m *StdMapper) ToJson(
	_*struct{
		miruken.Maps
		miruken.Format `to:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions api.PolymorphicOptions,
	ctx miruken.HandleContext,
) (js string, err error) {
	var data []byte
	src := maps.Source()
	if polyOptions.PolymorphicHandling == miruken.Set(api.PolymorphicHandlingRoot) {
		src = &typeContainer{src, ctx.Composer()}
	}
	if transformers := options.Transformers; len(transformers) > 0 {
		src = &transformer{src, transformers}
	}
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(src, prefix, indent)
	} else {
		data, err = json.Marshal(src)
	}
	return string(data), err
}

func (m *StdMapper) ToJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `to:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions api.PolymorphicOptions,
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
		if polyOptions.PolymorphicHandling == miruken.Set(api.PolymorphicHandlingRoot) {
			src = &typeContainer{src, ctx.Composer()}
		}
		if transformers := options.Transformers; len(transformers) > 0 {
			src = &transformer{src, transformers}
		}
		err    = enc.Encode(src)
		stream = *writer
	}
	return stream, err
}

func (m *StdMapper) FromJson(
	_*struct{
	    miruken.Maps
		miruken.Format `from:"application/json"`
	  }, jsonString string,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions api.PolymorphicOptions,
	maps *miruken.Maps,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	if transformers := options.Transformers; len(transformers) > 0 {
		t := transformer{target, transformers}
		target = &t
	}
	if polyOptions.PolymorphicHandling == miruken.Set(api.PolymorphicHandlingRoot) {
		tc := typeContainer{target, ctx.Composer()}
		err := json.Unmarshal([]byte(jsonString), &tc)
		return target, err
	}
	err := json.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *StdMapper) FromJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `from:"application/json"`
	  }, stream io.Reader,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions api.PolymorphicOptions,
	maps *miruken.Maps,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	dec    := json.NewDecoder(stream)
	if transformers := options.Transformers; len(transformers) > 0 {
		t := transformer{target, transformers}
		target = &t
	}
	if polyOptions.PolymorphicHandling == miruken.Set(api.PolymorphicHandlingRoot) {
		tc := typeContainer{target,ctx.Composer()}
		err := dec.Decode(&tc)
		return target, err
	}
	err := dec.Decode(target)
	return target, err
}


// Format returns a miruken.Builder for controlling indentation and formatting.
func Format(prefix, indent string) miruken.Builder {
	return miruken.Options(StdOptions{Prefix: prefix, Indent: indent})
}

// Transform returns a miruken.Builder that applies all transformations.
func Transform(transformers ...transform.Transformer) miruken.Builder {
	return miruken.Options(StdOptions{Transformers: transformers})
}

type (
	// typeContainer is a helper type used to emit type field
	// information for polymorphic serialization/deserialization.
	typeContainer struct {
		v        any
		composer miruken.Handler
	}

	// transformer applies all transformations to the json bytes.
	transformer struct {
		v            any
		transformers []transform.Transformer
	}
)


// typeContainer

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
			elem   := typeContainer{s.Index(i).Interface(), c.composer}
			if err := enc.Encode(&elem); err != nil {
				return nil, fmt.Errorf("can't marshal array index %d: %w", i, err)
			} else {
				raw := json.RawMessage(b.Bytes())
				arr = append(arr, &raw)
			}
		}
		v = arr
	}
	if byt, err := json.Marshal(v); err != nil {
		return nil, err
	} else if len(byt) > 0 && byt[0] == '{' {
		typeInfo, _, err := miruken.Map[TypeFieldInfo](c.composer, v, ToTypeInfo)
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
						tc  := typeContainer{&target,c.composer}
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
			return err
		} else if err := json.Unmarshal(data, v); err != nil {
			return err
		} else {
			if late, ok := c.v.(*miruken.Late); ok {
				late.Value = v
			} else {
				miruken.CopyIndirect(v, c.v)
			}
		}
	}
	return nil
}


// transformer

func (t *transformer) MarshalJSON() ([]byte, error) {
	container := conjson.NewMarshaler(t.v, t.transformers...)
	return json.Marshal(container)
}

func (t *transformer) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(
		data,
		conjson.NewUnmarshaler(t.v, t.transformers...),
	)
}