package json

import (
	"github.com/json-iterator/go"
	"github.com/miruken-go/miruken"
	"io"
)

// IterMapper formats to and from json using https://github.com/json-iterator/go.
type (
	IterMapper struct{}

	IterOptions struct {
		Config jsoniter.Config
		Api    jsoniter.API
	}
)

func (m *IterMapper) ToJson(
	_*struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options IterOptions,
) (string, error) {
	api := effectiveApi(&options)
	return api.MarshalToString(maps.Source())
}

func (m *IterMapper) ToJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options IterOptions,
) (stream io.Writer, err error) {
	if writer, ok := maps.Target().(*io.Writer); ok && !miruken.IsNil(writer) {
		api := effectiveApi(&options)
		enc := api.NewEncoder(*writer)
		err    = enc.Encode(maps.Source())
		stream = *writer
	}
	return stream, err
}

func (m *IterMapper) FromJson(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, jsonString string,
	maps *miruken.Maps,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options IterOptions,
) (any, error) {
	api    := effectiveApi(&options)
	target := maps.Target()
	err    := api.Unmarshal([]byte(jsonString), target)
	return target, err
}

func (m *IterMapper) FromJsonStream(
	_*struct{
	    miruken.Maps
		miruken.Format `as:"application/json"`
	  }, stream io.Reader,
	maps *miruken.Maps,
	_*struct{
		miruken.Optional
		miruken.FromOptions
	  }, options IterOptions,
) (any, error) {
	api    := effectiveApi(&options)
	target := maps.Target()
	dec    := api.NewDecoder(stream)
	err    := dec.Decode(target)
	return target, err
}

func effectiveApi(options *IterOptions) jsoniter.API {
	if api := options.Api; miruken.IsNil(api) {
		if config := &options.Config; (jsoniter.Config{}) != *config {
			return config.Froze()
		}
	}
	return jsoniter.ConfigDefault
}