package jsonstd

import (
	"bytes"
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
	// Options provide options for controlling json encoding.
	Options struct {
		Prefix       string
		Indent       string
		EscapeHTML   miruken.Option[bool]
		Transformers []transform.Transformer
	}

	// Mapper formats to and from json using encoding/json.
	Mapper struct{}
)

func (m *Mapper) ToJson(
	_*struct{
		maps.Format `to:"application/json"`
	  }, it *maps.It,
	_*struct{
		args.Optional
		args.FromOptions
	  }, options Options,
	_*struct{
		args.Optional
		args.FromOptions
	  }, apiOptions api.Options,
	ctx miruken.HandleContext,
) (any, error) {
	switch t := it.Target().(type) {
	case *[]byte:
		return marshal(it, &options, &apiOptions, ctx)
	case *io.Writer:
		var writer io.Writer
		if miruken.IsNil(*t) {
			var b bytes.Buffer
			w := io.Writer(&b)
			writer = w
		} else {
			writer = *t
		}
		err := encode(it, writer, &options, &apiOptions, ctx)
		if err == nil {
			return writer, nil
		}
		return nil, err
	default:
		byt, err := marshal(it, &options, &apiOptions, ctx)
		if err != nil {
			return nil, err
		}
		t = it.Target()
		if _, err = maps.Into(ctx.Composer(), byt, &t); err != nil {
			return nil, err
		}
		return t, nil
	}
	//return nil, nil
}

func (m *Mapper) FromBytes(
	_*struct{
		maps.Format `from:"application/json"`
	  }, byt []byte,
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
	return unmarshal(maps, byt, &options, &apiOptions, ctx)
}

func (m *Mapper) FromReader(
	_*struct{
		maps.Format `from:"application/json"`
	  }, reader io.Reader,
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
	return decode(maps, reader, &options, &apiOptions, ctx)
}

/*
func (m *Mapper) FromString(
	_*struct{
		maps.Format `from:"application/json"`
	  }, json string,
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
	return unmarshal(maps, []byte(json), &options, &apiOptions, ctx)
}
*/

func marshal(
	maps       *maps.It,
	options    *Options,
	apiOptions *api.Options,
	ctx        miruken.HandleContext,
) (byt []byte, err error) {
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
		byt, err = json.MarshalIndent(src, prefix, indent)
	} else {
		byt, err = json.Marshal(src)
	}
	return
}

func unmarshal(
	maps       *maps.It,
	byt        []byte,
	options    *Options,
	apiOptions *api.Options,
	ctx        miruken.HandleContext,
) (target any, err error) {
	target = maps.Target()
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		tc := typeContainer{
			v:        target,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
		err = json.Unmarshal(byt, &tc)
		return
	} else if trans := options.Transformers; len(trans) > 0 {
		t := transformer{target, trans}
		target = &t
	}
	err = json.Unmarshal(byt, target)
	return
}

func encode(
	maps       *maps.It,
	writer     io.Writer,
	options    *Options,
	apiOptions *api.Options,
	ctx        miruken.HandleContext,
) error {
	enc := json.NewEncoder(writer)
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
	return enc.Encode(src)
}

func decode(
	maps       *maps.It,
	reader     io.Reader,
	options    *Options,
	apiOptions *api.Options,
	ctx        miruken.HandleContext,
) (target any, err error) {
	target = maps.Target()
	dec := json.NewDecoder(reader)
	if apiOptions.Polymorphism == miruken.Set(api.PolymorphismRoot) {
		tc := typeContainer{
			v:        target,
			trans:    options.Transformers,
			composer: ctx.Composer(),
		}
		err = dec.Decode(&tc)
		return
	} else if trans := options.Transformers; len(trans) > 0 {
		t := transformer{target, trans}
		target = &t
	}
	err = dec.Decode(target)
	return
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