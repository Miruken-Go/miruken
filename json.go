package miruken

import (
	"encoding/json"
	"io"
)

type (
	// JsonMapper formats to and from json.
	JsonMapper struct{}

	// JsonOptions customizes json formatting.
	JsonOptions struct {
		Prefix string
		Indent string
	}
)

func (m *JsonMapper) ToJson(
	_ *struct{
		Maps
		Format `as:"application/json"`
	  }, maps *Maps,
	_ *struct{
		Optional
		FromOptions
	  }, options JsonOptions,
) (js string, err error) {
	var data []byte
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(maps.Source(), prefix, indent)
	} else {
		data, err = json.Marshal(maps.Source())
	}
	return string(data), err
}

func (m *JsonMapper) ToJsonStream(
	_ *struct{
		Maps
		Format `as:"application/json"`
	  }, maps *Maps,
	_ *struct{
		Optional
		FromOptions
	  }, options JsonOptions,
) (stream io.Writer, err error) {
	if writer, ok := maps.target.(*io.Writer); ok && !IsNil(writer) {
		enc := json.NewEncoder(*writer)
		if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
			enc.SetIndent(prefix, indent)
		}
		err    = enc.Encode(maps.Source())
		stream = *writer
	}
	return stream, err
}

func (m *JsonMapper) FromJson(
	_ *struct{
		Maps
		Format `as:"application/json"`
	  }, jsonString string,
	maps *Maps,
) (any, error) {
	target := maps.Target()
	err    := json.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *JsonMapper) FromJsonStream(
	_ *struct{
		Maps
		Format `as:"application/json"`
	  }, stream io.Reader,
	maps *Maps,
) (any, error) {
	target := maps.Target()
	dec    := json.NewDecoder(stream)
	err    := dec.Decode(target)
	return target, err
}