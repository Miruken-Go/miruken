package api

import (
	"fmt"
	"mime"
	"net/textproto"
	"reflect"

	"github.com/miruken-go/miruken/maps"
)

// Content is information produced or consumed by an api.
type Content interface {
	MediaType() string
	Metadata() map[string]any
	Body() any
	// optional WriteBody() any
}

var (
	// ToJson encodes a model into json format
	ToJson = maps.To("application/json", nil)

	// FromJson decodes json into a corresponding model
	FromJson = maps.From("application/json", nil)
)

// ParseMediaType parses the mediaType into a maps.Format suitable
// for mapping in the requested direction.
func ParseMediaType(
	mediaType string,
	direction maps.Direction,
) (*maps.Format, error) {
	if mt, params, err := mime.ParseMediaType(mediaType); err != nil {
		return nil, err
	} else if direction == maps.DirectionTo {
		return maps.To(mt, params), nil
	} else {
		return maps.From(mt, params), nil
	}
}

// FormatMediaType formats the maps.Format into a media type conforming
// to RFC 2045 and RFC 2616.
func FormatMediaType(format *maps.Format) string {
	if format == nil {
		panic("format cannot be nil")
	}
	switch format.Rule() {
	case maps.FormatRuleEquals:
		return mime.FormatMediaType(format.Name(), format.Params())
	case maps.FormatRuleStartsWith:
		return mime.FormatMediaType(format.Name()+"/*", format.Params())
	default:
		return ""
	}
}

// NewHeader creates a mime header from the supplied key values.
func NewHeader(
	metadata map[string]any,
) textproto.MIMEHeader {
	header := textproto.MIMEHeader{}
	MergeHeader(header, metadata)
	return header
}

// MergeHeader merges the supplied key values into the existing mime header.
func MergeHeader(
	header   textproto.MIMEHeader,
	metadata map[string]any,
) {
	if header == nil {
		panic("header cannot be nil")
	}
	for k, v := range metadata {
		typ := reflect.TypeOf(v)
		switch typ.Kind() {
		case reflect.Slice, reflect.Array:
			vs := reflect.ValueOf(v)
			for i := range vs.Len() {
				header.Add(k, fmt.Sprintf("%v", vs.Index(i)))
			}
		default:
			header.Set(k, fmt.Sprintf("%v", v))
		}
	}
}
