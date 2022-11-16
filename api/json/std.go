package json

import (
	"encoding/json"
	"github.com/miruken-go/miruken"
	"io"
)

// StdMapper formats to and from json using encoding/json.
type (
	StdMapper struct{}

	StdOptions struct {
		Prefix            string
		Indent            string
		TypeFieldHandling TypeFieldHandling
	}
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
) (js string, err error) {
	var data []byte
	if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
		data, err = json.MarshalIndent(maps.Source(), prefix, indent)
	} else {
		data, err = json.Marshal(maps.Source())
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
) (stream io.Writer, err error) {
	if writer, ok := maps.Target().(*io.Writer); ok && !miruken.IsNil(writer) {
		enc := json.NewEncoder(*writer)
		if prefix, indent := options.Prefix, options.Indent; len(prefix) > 0 || len(indent) > 0 {
			enc.SetIndent(prefix, indent)
		}
		err    = enc.Encode(maps.Source())
		stream = *writer
	}
	return stream, err
}

func (m *StdMapper) FromJson(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, jsonString string,
	maps *miruken.Maps,
) (any, error) {
	target := maps.Target()
	err    := json.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *StdMapper) FromJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, stream io.Reader,
	maps *miruken.Maps,
) (any, error) {
	target := maps.Target()
	dec    := json.NewDecoder(stream)
	err    := dec.Decode(target)
	return target, err
}