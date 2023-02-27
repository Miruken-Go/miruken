package api

import (
	"bytes"
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
		maps.It
		maps.Format `from:"/multipart//"`
	  }, reader *multipart.Reader,
) (PartContainer, error) {
	var mb ReadPartsBuilder
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		header := p.Header
		key    := p.FormName()

		if key == "" {
			if key = header.Get("Content-ID"); key == "" {
				continue
			}
		}

		content, err := io.ReadAll(p)
		if err != nil {
			return nil, err
		}

		mb.AddPart(key, (&PartBuilder{}).
			ContentType(header.Get("Content-Type")).
			Metadata(header).
			Content(bytes.NewReader(content)).
			Filename(p.FileName()).
			Build())
	}
	return mb.Build(), nil
}
