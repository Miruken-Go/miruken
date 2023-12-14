package api

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/miruken-go/miruken/internal"
)

type (
	// Part represents a piece of a message.
	// Part can be used in a PartContainer or
	// independently to control message encoding.
	Part interface {
		Content
		Filename() string
	}

	// PartContainer stores all the Part's of a message.
	// The main part is typically the message payload.
	PartContainer interface {
		Content
		Parts() map[string][]Part
		MainPart() Part
	}

	// PartBuilder builds a message Part.
	PartBuilder struct {
		part part
	}

	// ReadPartsBuilder builds a PartContainer for reading Part's.
	ReadPartsBuilder struct {
		container partContainer
	}

	// WritePartsBuilder builds a PartContainer for writing Part's.
	WritePartsBuilder struct {
		ReadPartsBuilder
	}

	part struct {
		mediaType string
		metadata  map[string]any
		filename  string
		body      any
	}

	partContainer struct {
		mediaType string
		boundary  string
		metadata  map[string]any
		parts     map[string][]Part
		main      Part
	}
)

// PartBuilder

func (b *PartBuilder) MediaType(
	mediaType string,
) *PartBuilder {
	if _, _, err := mime.ParseMediaType(mediaType); err != nil {
		panic(fmt.Errorf("invalid media type %q (%w)", mediaType, err))
	}
	b.part.mediaType = mediaType
	return b
}

func (b *PartBuilder) Metadata(
	metadata map[string]any,
) *PartBuilder {
	b.part.metadata = metadata
	return b
}

func (b *PartBuilder) MetadataStrings(
	metadata map[string][]string,
) *PartBuilder {
	meta := make(map[string]any)
	for k, val := range metadata {
		if len(val) == 1 {
			meta[k] = val[0]
		} else {
			meta[k] = val
		}
	}
	return b.Metadata(meta)
}

func (b *PartBuilder) Filename(
	filename string,
) *PartBuilder {
	if filename != "" {
		filename = filepath.Base(filename)
	}
	b.part.filename = filename
	return b
}

func (b *PartBuilder) Body(
	body any,
) *PartBuilder {
	if internal.IsNil(body) {
		panic("body cannot be nil")
	}
	b.part.body = body
	return b
}

func (b *PartBuilder) Build() Part {
	part := b.part
	if part.metadata == nil {
		part.metadata = make(map[string]any)
	}
	return part
}

// ReadPartsBuilder

func (b *ReadPartsBuilder) Metadata(
	metadata map[string]any,
) *ReadPartsBuilder {
	b.container.metadata = metadata
	return b
}

func (b *ReadPartsBuilder) MainPart(
	main Part,
) *ReadPartsBuilder {
	b.container.main = main
	return b
}

func (b *ReadPartsBuilder) AddPart(
	key string,
	part Part,
) *ReadPartsBuilder {
	if len(key) == 0 {
		panic("key cannot be empty")
	}
	if internal.IsNil(part) {
		panic("part cannot be nil")
	}
	parts := b.container.parts
	if parts == nil {
		parts = make(map[string][]Part)
		b.container.parts = map[string][]Part{
			key: {part},
		}
	} else if set, ok := parts[key]; ok {
		parts[key] = append(set, part)
	} else {
		parts[key] = []Part{part}
	}
	return b
}

func (b *ReadPartsBuilder) AddParts(
	parts map[string][]Part,
) *ReadPartsBuilder {
	for key, set := range parts {
		for _, part := range set {
			b.AddPart(key, part)
		}
	}
	return b
}

func (b *ReadPartsBuilder) NewPart() *PartBuilder {
	return &PartBuilder{}
}

func (b *ReadPartsBuilder) Build() PartContainer {
	ctr := b.container
	if ctr.parts == nil {
		ctr.parts = make(map[string][]Part)
	}
	if ctr.metadata == nil {
		ctr.metadata = make(map[string]any)
	}
	return ctr
}

// WritePartsBuilder

func (b *WritePartsBuilder) MediaType(
	mediaType string,
) *WritePartsBuilder {
	if _, params, err := mime.ParseMediaType(mediaType); err != nil {
		panic(fmt.Errorf("invalid media type %q: %w", mediaType, err))
	} else {
		b.container.mediaType = mediaType
		b.container.boundary = params["boundary"]
	}
	return b
}

func (b *WritePartsBuilder) Build() PartContainer {
	ctr := &b.container
	if ctr.parts == nil {
		ctr.parts = make(map[string][]Part)
	}
	if ctr.metadata == nil {
		ctr.metadata = make(map[string]any)
	}
	if len(ctr.mediaType) == 0 {
		ctr.mediaType = "multipart/form-data"
	} else if !strings.HasPrefix(ctr.mediaType, "multipart/") {
		ctr.mediaType = "multipart/" + ctr.mediaType
	}
	if len(ctr.boundary) == 0 {
		ctr.mediaType = ctr.mediaType + "; boundary=" + randomBoundary()
	}
	return ctr
}

// part

func (p part) MediaType() string {
	return p.mediaType
}

func (p part) Metadata() map[string]any {
	return p.metadata
}

func (p part) Filename() string {
	return p.filename
}

func (p part) Body() any {
	return p.body
}

// partContainer

func (c partContainer) MediaType() string {
	return c.mediaType
}

func (c partContainer) Metadata() map[string]any {
	return c.metadata
}

func (c partContainer) Parts() map[string][]Part {
	return c.parts
}

func (c partContainer) MainPart() Part {
	return c.main
}

func (c partContainer) Body() any {
	if main := c.main; main != nil {
		return main.Body()
	}
	return nil
}

func (c partContainer) WriteBody() any {
	return c
}

// randomBoundary copied from multipart.randomBoundary
func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
