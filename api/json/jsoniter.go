package json

import (
	"fmt"
	"github.com/json-iterator/go"
	"github.com/miruken-go/miruken"
	"github.com/modern-go/reflect2"
	"io"
	"reflect"
	"unsafe"
)

// IterMapper formats to and from json using https://github.com/json-iterator/go.
type (
	IterMapper struct{}

	IterOptions struct {
		Config jsoniter.Config
		Api    jsoniter.API
	}
)

// IterMapper

func (m *IterMapper) ToJson(
	_*struct{
		miruken.Maps
		miruken.Format `as:"application/json"`
	  }, maps *miruken.Maps,
	_*struct{
	    miruken.Optional
	    miruken.FromOptions
	  }, options IterOptions,
	ctx miruken.HandleContext,
) (string, error) {
	api := effectiveApi(&options)
	val := valCtx{maps.Source(), ctx.Composer()}
	return api.MarshalToString(val)
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
		api   := effectiveApi(&options)
		enc   := api.NewEncoder(*writer)
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
			api := config.Froze()
			api.RegisterExtension(&polymorphicExtension{
				jsoniter.EncoderExtension{
					reflect2.TypeOf(valCtx{}): &valCtxEncoder{},
				}, nil,
			})
			api.EncoderOf(reflect2.TypeOf(typeIdDummy{}))
			return api
		}
	}
	return jsoniter.ConfigDefault
}

type (
	valCtx struct {
		val      any
		composer miruken.Handler
	}

	valCtxEncoder struct {}
	valCtxDecoder struct {}

	typeIdDummy struct {
		TypeId  string `json:"$type"`
	}

	typeIdEncoder struct {}

	polymorphicExtension struct {
		jsoniter.EncoderExtension
		typeId *jsoniter.Binding
	}
)

// valCtxEncoder

func (v *valCtxEncoder) Encode(
	ptr     unsafe.Pointer,
	stream *jsoniter.Stream,
) {
	ctx := (*valCtx)(ptr)
	val := ctx.val
	stream.Attachment = ctx
	stream.WriteVal(val)
}

func (v *valCtxEncoder) IsEmpty(
	ptr unsafe.Pointer,
) bool {
	return false
}

// Polymorphism

func (t typeIdEncoder) Encode(
	ptr     unsafe.Pointer,
	stream *jsoniter.Stream,
) {
	if ctx, ok := stream.Attachment.(*valCtx); ok {
		if !miruken.IsNil(ctx.val) {
			fmt.Printf("%v: %p\n", reflect.TypeOf(ctx.val).String(), ptr)
			ctx.val = nil
		}
	}
}

func (t typeIdEncoder) IsEmpty(
	ptr unsafe.Pointer,
) bool {
	return true
}

func (p *polymorphicExtension) UpdateStructDescriptor(
	structDescriptor *jsoniter.StructDescriptor,
) {
	if structDescriptor.Type == _typeIdType {
		p.typeId = structDescriptor.Fields[0]
		p.typeId.Encoder = typeIdEncoder{}
	}
}

var _typeIdType = reflect2.TypeOf(typeIdDummy{})