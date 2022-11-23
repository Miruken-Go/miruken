package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"io"
	"reflect"
)

// StdMapper formats to and from json using encoding/json.
type (
	StdMapper struct{}

	StdOptions struct {
		Prefix            string
		Indent            string
		EscapeHTML        miruken.Option[bool]
		TypeFieldHandling miruken.Option[TypeFieldHandling]
	}
)

var (
	AsApplicationJson = miruken.As("application/json")
)


// StdMapper

func (m *StdMapper) ToJson(
	_*struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions PolymorphicOptions,
	ctx miruken.HandleContext,
) (js string, err error) {
	var data []byte
	src := maps.Source()
	if options.TypeFieldHandling == miruken.Set(TypeFieldHandlingRoot) {
		src = &typeContainer{src, polyOptions.KnownTypeFields, ctx.Composer()}
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
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions PolymorphicOptions,
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
		if options.TypeFieldHandling == miruken.Set(TypeFieldHandlingRoot) {
			src = &typeContainer{src, polyOptions.KnownTypeFields,ctx.Composer()}
		}
		err    = enc.Encode(src)
		stream = *writer
	}
	return stream, err
}

func (m *StdMapper) FromJson(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, jsonString string,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions PolymorphicOptions,
	maps *miruken.Maps,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	if options.TypeFieldHandling == miruken.Set(TypeFieldHandlingRoot) {
		tc := typeContainer{target, polyOptions.KnownTypeFields, ctx.Composer()}
		err := json.Unmarshal([]byte(jsonString), &tc)
		return tc.v, err
	}
	err := json.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *StdMapper) FromJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, stream io.Reader,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options StdOptions,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, polyOptions PolymorphicOptions,
	maps *miruken.Maps,
	ctx  miruken.HandleContext,
) (any, error) {
	target := maps.Target()
	dec    := json.NewDecoder(stream)
	if options.TypeFieldHandling == miruken.Set(TypeFieldHandlingRoot) {
		tc := typeContainer{target, polyOptions.KnownTypeFields, ctx.Composer()}
		err := dec.Decode(&tc)
		return tc.v, err
	}
	err := dec.Decode(target)
	return target, err
}

type (
	// typeContainer is a helper type used to emit type field
	// information for polymorphic serialization/deserialization.
	typeContainer struct {
		v         any
		fields    []string
		composer  miruken.Handler
	}
)

func (c *typeContainer) MarshalJSON() ([]byte, error) {
	v := c.v
	if byt, err := json.Marshal(v); err != nil {
		return nil, err
	} else {
		if typ := reflect.TypeOf(v); typ != nil && typ.Kind() == reflect.Struct {
			typeInfo, _, err := miruken.Map[TypeFieldInfo](c.composer, v)
			if err != nil {
				return nil, err
			}
			typeProperty := []byte(fmt.Sprintf("\"%v\":\"%v\",", typeInfo.Field, typeInfo.Value))
			byt = append(byt, typeProperty...)
			copy(byt[len(typeProperty)+1:], byt[1:])
			copy(byt[1:], typeProperty)
		}
		return byt, nil
	}
}

func (c *typeContainer) UnmarshalJSON(data []byte) error {
	var fields map[string]*json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	var field string
	var typeIdRaw *json.RawMessage
	for _, field = range c.fields {
		if typeIdRaw = fields[field]; typeIdRaw != nil {
			break
		}
	}
	if typeIdRaw == nil {
		return errors.New("missing type field")
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
			c.v = v
		}
	}
	return nil
}
