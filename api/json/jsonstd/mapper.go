package jsonstd

import (
	"encoding/json"
	"github.com/Rican7/conjson"
	"github.com/Rican7/conjson/transform"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/maps"
	"io"
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
		args.Optional
		args.FromOptions
	  }, options Options,
	_*struct{
		args.Optional
		args.FromOptions
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
		args.Optional
		args.FromOptions
	  }, options Options,
	_*struct{
		args.Optional
		args.FromOptions
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
		args.Optional
		args.FromOptions
	  }, options Options,
	_*struct{
		args.Optional
		args.FromOptions
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
		args.Optional
		args.FromOptions
	  }, options Options,
	_*struct{
		args.Optional
		args.FromOptions
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