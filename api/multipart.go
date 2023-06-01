package api

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/maps"
	"io"
	"mime/multipart"
	"time"
)

// MultipartMapper reads and writes 'multipart/*'
// mime messages from a PartContainer.
type MultipartMapper struct{}


func (m *MultipartMapper) Read(
	_*struct{
		maps.Format `from:"/multipart//"`
	  }, reader io.Reader,
	  it  *maps.It,
	  ctx miruken.HandleContext,
) (Message, error) {
	var msg Message
	_, boundary, start := extractMultipartParams(it)
	if boundary == "" {
		return msg, ErrMissingBoundary
	}

	var main Part
	var rpb ReadPartsBuilder
	composer := ctx.Composer()
	now := time.Now().UnixNano()
	mr  := multipart.NewReader(reader, boundary)

	for i := 0;; i++ {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return msg, err
		}

		body, err := io.ReadAll(p)
		if err != nil {
			return msg, err
		}

		addPart := true
		header  := p.Header
		ct      := header.Get("Content-Type")

		filename := p.FileName()
		if filename == "" {
			filename = header.Get("Content-Filename")
		}
		var pb PartBuilder
		pb.MediaType(ct).
		   MetadataStrings(header).
		   Filename(filename)

		var key string
		if key = p.FormName(); key == "" {
			key = header.Get("Content-ID")
		}
		if key == "" {
			key = fmt.Sprintf("%d-%d", now, i)
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
				reader := bytes.NewReader(body)
				rpb.AddPart(key, pb.Body(reader).Build())
			}
		} else if len(body) > 0 {
			late, _, _, err := maps.Out[miruken.Late](composer, body, maps.From(ct, nil))
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
				main = pb.Body(payload).Build()
				rpb.MainPart(main)
			}
		} else {
			main = pb.Build()
			rpb.MainPart(main)
		}
	}
	msg.Payload = rpb.Build()
	return msg, nil
}

func (m *MultipartMapper) Write(
	_*struct{
		maps.Format `to:"/multipart//"`
	  }, msg Message,
	it  *maps.It,
	ctx miruken.HandleContext,
) (io.Writer, error) {
	if parts, ok := msg.Payload.(PartContainer); ok {
		return m.WriteParts(nil, parts, it, ctx)
	}
	return nil, nil
}

func (m *MultipartMapper) WriteParts(
	_*struct{
		maps.Format `to:"/multipart//"`
	  }, pc PartContainer,
	it  *maps.It,
	ctx miruken.HandleContext,
) (io.Writer, error) {
	if writer, ok := it.Target().(*io.Writer); ok {
		typ, boundary, start := extractMultipartParams(it)
		if boundary == "" {
			return nil, ErrMissingBoundary
		}
		if start == "" {
			start = "main"
		}
		mw := multipart.NewWriter(*writer)
		if err := mw.SetBoundary(boundary); err != nil {
			return nil, err
		}
		if main := pc.MainPart(); main != nil {
			if err := addPart(start, typ, main, mw, ctx); err != nil {
				return nil, err
			}
		}
		for key, set := range pc.Parts() {
			for _, part := range set {
				if err := addPart(key, typ, part, mw, ctx); err != nil {
					return nil, err
				}
			}
		}
		if err := mw.Close(); err != nil {
			return nil, err
		}
		return *writer, nil
	}
	return nil, nil
}

func addPart(
	key    string,
	typ    string,
	part   Part,
	writer *multipart.Writer,
	ctx    miruken.HandleContext,
) error {
	contentType := part.MediaType()
	header := NewHeader(part.Metadata())
	header.Set("Content-Type", contentType)

	if typ == "multipart/form-data" {
		if header.Get("Content-Disposition") == "" {
			if filename := part.Filename(); filename != "" {
				header.Set("Content-Disposition",
					fmt.Sprintf(`form-data; name="%s"; filename="%s"`, key, filename))
			} else {
				header.Set("Content-Disposition",
					fmt.Sprintf(`form-data; name="%s"`, key))
			}
		}
	} else {
		if header.Get("Content-ID") == "" {
			header.Set("Content-ID", key)
		}
		if filename := part.Filename(); filename != "" {
			header.Set("Content-Filename", filename)
		}
	}

	w, err := writer.CreatePart(header)
	if err == nil {
		body := part.Body()
		if reader, ok := body.(io.Reader); ok {
			_, err = io.Copy(w, reader)
		} else {
			var format *maps.Format
			format, err = ParseMediaType(contentType, maps.DirectionTo)
			if err == nil {
				_, _, err = maps.Into(ctx.Composer(), body, &w, format)
			}
		}
	}
	return err
}

func extractMultipartParams(
	src miruken.ConstraintSource,
) (typ string, boundary string, start string) {
	if format, ok := constraints.First[*maps.Format](src); ok {
		if b, ok := format.Params()["boundary"]; ok {
			boundary = b
		}
		if s, ok := format.Params()["start"]; ok {
			start = s
		}
		typ = format.Name()
	}
	return
}

var ErrMissingBoundary = errors.New(`multipart: missing "boundary" parameter`)