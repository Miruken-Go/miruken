package api

import (
	"bytes"
	"errors"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/maps"
	"io"
	"mime/multipart"
)

type (
	// MultipartMapper maps 'multipart/*' mime messages to
	// a corresponding PartContainer for reading and writing.
	MultipartMapper struct{}
)

func (m *MultipartMapper) Read(
	_*struct{
		maps.Format `from:"/multipart//"`
	  }, reader io.Reader,
	  it *maps.It,
	  ctx miruken.HandleContext,
) (Message, error) {
	var msg Message
	boundary, start := extractMultipartParams(it)
	if boundary == "" {
		return msg, errors.New("multipart: missing \"boundary\" parameter")
	}
	var main Part
	composer := ctx.Composer()
	mr := multipart.NewReader(reader, boundary)
	var mb ReadPartsBuilder
	for i := 0;; i++ {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return msg, err
		}

		content, err := io.ReadAll(p)
		if err != nil {
			return msg, err
		}

		addPart := true
		header  := p.Header
		ct      := header.Get("Content-Type")

		var pb PartBuilder
		pb.ContentType(ct).
			Metadata(header).
			Filename(p.FileName())

		var key string
		if key = p.FormName(); key == "" {
			key = header.Get("Content-ID")
		}

		if main == nil {
			if start == "" {
				addPart = i > 0
			} else {
				addPart = key != start
			}
		}

		if addPart {
			if key != "" {
				reader := bytes.NewReader(content)
				mb.AddPart(key, pb.Content(reader).Build())
			}
		} else if len(content) > 0 {
			late, _, err := maps.Map[miruken.Late](composer, content, maps.From(ct, nil))
			if err != nil {
				return msg, err
			}
			if sur, ok := late.Value.(Surrogate); ok {
				late.Value, err = sur.Original(composer)
				if err != nil {
					return msg, err
				}
			}
			if payload := late.Value; payload != nil {
				main = pb.Content(payload).Build()
				mb.MainPart(main)
			}
		} else {
			main = pb.Build()
			mb.MainPart(main)
		}
	}
	msg.Payload = mb.Build()
	return msg, nil
}

func extractMultipartParams(
	src miruken.ConstraintSource,
) (boundary string, start string) {
	if format, ok := constraints.First[*maps.Format](src); ok {
		if b, ok := format.Params()["boundary"]; ok {
			boundary = b
		}
		if s, ok := format.Params()["start"]; ok {
			start = s
		}
	}
	return
}